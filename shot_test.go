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
// so two loaders never shape text at the same time.
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
	waitForRequests(t, 1) // the channel list has arrived

	shellWin := test.NewWindow(layout)
	shellWin.Resize(fyne.NewSize(1000, 680))
	time.Sleep(300 * time.Millisecond)

	snap := func(name string) {
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

	snap("shot-shell-empty.png")

	openChat(Channel{
		ID:          "11111111-1111-4111-8111-111111111111",
		ChannelName: "general",
		ChannelType: "group",
	})
	waitForRequests(t, 2) // the message history has arrived
	snap("shot-shell-chat.png")
}
