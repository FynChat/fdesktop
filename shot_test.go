//go:build screenshot

// Renders the views offscreen with the Fyne software driver and writes PNGs,
// so the UI can be reviewed without opening a window.
//
// It is behind a build tag on purpose. Fyne's test driver runs fyne.Do inline
// instead of queueing it to the main thread, so the goroutines these views
// start really do touch the UI concurrently with the test: the race detector
// fires even though the production code is correct.

package main

import (
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func fakeBackend() *httptest.Server {
	fakeHits.Store(0)
	record := func() { fakeHits.Add(1) }

	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		record()
		w.Write([]byte(`{"token":"t","user":{"id":1,"name":"Ada","username":"ada"}}`))
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		record()
		w.Write([]byte(`{"token":"t","user":{"id":1,"name":"Ada","username":"ada"}}`))
	})
	mux.HandleFunc("/api/channels", func(w http.ResponseWriter, r *http.Request) {
		record()
		w.Write([]byte(`[
			{"id":"11111111-1111-4111-8111-111111111111","channel_name":"general","channel_avatar":"","channel_type":"group"},
			{"id":"22222222-2222-4222-8222-222222222222","channel_name":"design-team","channel_avatar":"","channel_type":"channel"},
			{"id":"33333333-3333-4333-8333-333333333333","channel_name":"ada","channel_avatar":"","channel_type":"p2p"}
		]`))
	})
	mux.HandleFunc("/api/channel/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"channel_name":"general","channel_avatar":""}`))
	})
	mux.HandleFunc("/api/channel/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"channel_name":"general","channel_avatar":""}`))
	})
	mux.HandleFunc("/api/contacts", func(w http.ResponseWriter, r *http.Request) {
		record()
		w.Write([]byte(`[
			{"id":2,"name":"Grace Hopper","username":"amazing","avatar_url":""},
			{"id":3,"name":"Alan Turing","username":"turing","avatar_url":""},
			{"id":4,"name":"Katherine Johnson","username":"kj","avatar_url":""}
		]`))
	})
	mux.HandleFunc("/api/channels/", func(w http.ResponseWriter, r *http.Request) {
		record()
		w.Write([]byte(`[
			{"id":"m1","channel_id":"11111111-1111-4111-8111-111111111111","sender_id":2,"content":"morning everyone — the desktop build is finally green","created_at":"2026-10-10T12:00:00Z"},
			{"id":"m2","channel_id":"11111111-1111-4111-8111-111111111111","sender_id":2,"content":"CI passed on all three platforms","created_at":"2026-10-10T12:02:00Z"},
			{"id":"m3","channel_id":"11111111-1111-4111-8111-111111111111","sender_id":3,"content":"does the channel list still 404?","created_at":"2026-10-11T09:00:00Z"},
			{"id":"m4","channel_id":"11111111-1111-4111-8111-111111111111","sender_id":1,"content":"no — we added GET /api/channels yesterday","created_at":"2026-10-11T09:05:00Z"},
			{"id":"m5","channel_id":"11111111-1111-4111-8111-111111111111","sender_id":3,"content":"this one is deliberately long so you can see how a message wraps once it runs out of width instead of stretching all the way across the window","created_at":"2026-10-11T09:10:00Z"}
		]`))
	})
	return httptest.NewServer(mux)
}

// fakeHits counts requests, so the harness can wait for one loader to finish
// before starting the next. Fyne's font cache is shared, and two goroutines
// shaping text at once can crash the shaper — which only happens in the test
// driver, where fyne.Do runs inline instead of queueing to the main thread.
var fakeHits atomic.Int32

// waitForRequests blocks until the fake backend has seen at least n requests,
// then gives the loaders a moment to finish the widget updates their responses
// trigger. The shots target also pins GOMAXPROCS to 1, which is what keeps the
// two renders from genuinely overlapping — see the Makefile comment.
func waitForRequests(t *testing.T, n int32) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && fakeHits.Load() < n {
		time.Sleep(10 * time.Millisecond)
	}
	// The response has arrived; give the goroutine a moment to finish the
	// widget updates it triggers.
	time.Sleep(300 * time.Millisecond)
}

func capture(t *testing.T, name string, content fyne.CanvasObject, w, h float32) {
	t.Helper()
	win := test.NewWindow(content)
	win.Resize(fyne.NewSize(w, h))
	time.Sleep(500 * time.Millisecond)

	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, win.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", name)
	win.Close()
}

func TestRenderViews(t *testing.T) {
	srv := fakeBackend()
	defer srv.Close()

	old := baseURL
	baseURL = srv.URL
	defer func() { baseURL = old }()

	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(newTelegramTheme())

	// Warm the font stack before rendering anything. In a fresh test app the
	// first string measurement can crash inside walkString on a cold cache,
	// which has nothing to do with the view being captured and everything to
	// do with this being the process's first text.
	warm := test.NewWindow(widget.NewLabel("FynChat"))
	warm.Resize(fyne.NewSize(300, 100))
	warm.Canvas().Capture()
	warm.Close()

	client := NewHTTPClient()
	client.Token = "test-token"
	client.UserID = 1
	prefs := NewPrefs(app.Preferences())

	w := test.NewWindow(nil)
	defer w.Close()

	capture(t, "shot-login.png", loginView(client, w, func() {}, func() {}), 1000, 680)
	capture(t, "shot-register.png", registerView(client, w, func() {}, func() {}), 1000, 680)

	// The shell with nothing open yet, and with a chat open. One window for
	// both: the same content object cannot be shown in a second window.
	layout, openChat := shell(client, prefs, w, func() {})

	shellWin := test.NewWindow(layout)
	shellWin.Resize(fyne.NewSize(1000, 680))

	snap := func(name string) {
		time.Sleep(300 * time.Millisecond)
		f, err := os.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, shellWin.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", name)
	}

	// The channel list loader runs first and must finish before the chat
	// loader starts: two renders at once is what crashes the test driver.
	waitForRequests(t, 1)
	snap("shot-shell-empty.png")

	openChat(Channel{
		ID:          "11111111-1111-4111-8111-111111111111",
		ChannelName: "general",
		ChannelType: "group",
	})
	waitForRequests(t, 2) // the message history has arrived
	snap("shot-shell-chat.png")

	dialogWin := test.NewWindow(nil)
	defer dialogWin.Close()
	dialogWin.Resize(fyne.NewSize(1000, 680))

	// The two dialogs and the Contacts screen. Each goes in its own window:
	// a window shows one content object, and the dialogs are overlays on top.
	newChannelDialog(dialogWin, func(kind, name string, avatarData []byte) error { return nil })
	snapIn(t, dialogWin, "shot-dialog-channel.png")

	contactWin := test.NewWindow(nil)
	defer contactWin.Close()
	contactWin.Resize(fyne.NewSize(1000, 680))
	addContactDialog(contactWin, func(username string) error { return nil })
	snapIn(t, contactWin, "shot-dialog-contact.png")

	// The Contacts screen loads its list off a goroutine, so wait for the
	// request before capturing.
	contactsWin := test.NewWindow(contactsView(client, w, func(Channel) {}, func() {}))
	defer contactsWin.Close()
	contactsWin.Resize(fyne.NewSize(420, 680))
	waitForRequests(t, 3)
	snapIn(t, contactsWin, "shot-contacts.png")
}

// snapIn writes a window's canvas to a PNG.
func snapIn(t *testing.T, win fyne.Window, name string) {
	t.Helper()

	time.Sleep(300 * time.Millisecond)

	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, win.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", name)
}
