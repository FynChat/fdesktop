//go:build screenshot

// Renders each view offscreen with the Fyne software driver and writes PNGs,
// so the UI can be reviewed without opening a window.
//
//	go run -tags screenshot .   is not how this runs: use `make shots`.
//
// It is behind a build tag on purpose. Fyne's test driver runs fyne.Do inline
// instead of queueing it to the main thread, so the goroutines these views
// start really do touch the UI concurrently with the test: the race detector
// fires even though the production code is correct.

package main

import (
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func fakeBackend() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"token":"t","user":{"id":1,"name":"Ada","username":"ada"}}`))
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"token":"t","user":{"id":1,"name":"Ada","username":"ada"}}`))
	})
	mux.HandleFunc("/api/channels", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"id":"1bbf2de4-afd9-428e-abbc-18091cb6dd4e","channel_name":"general","channel_avatar":"","channel_type":"text"},
			{"id":"2bbf2de4-afd9-428e-abbc-18091cb6dd4e","channel_name":"random","channel_avatar":"","channel_type":"text"},
			{"id":"3bbf2de4-afd9-428e-abbc-18091cb6dd4e","channel_name":"design-team","channel_avatar":"","channel_type":"channel"}
		]`))
	})
	mux.HandleFunc("/api/channel/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"channel_name":"general","channel_avatar":""}`))
	})
	mux.HandleFunc("/api/channels/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"id":"m1","channel_id":"1bbf2de4-afd9-428e-abbc-18091cb6dd4e","sender_id":2,"content":"hey everyone","created_at":"2026-10-10T12:00:00Z"},
			{"id":"m2","channel_id":"1bbf2de4-afd9-428e-abbc-18091cb6dd4e","sender_id":2,"content":"are we still shipping tonight?","created_at":"2026-10-10T12:01:00Z"},
			{"id":"m3","channel_id":"1bbf2de4-afd9-428e-abbc-18091cb6dd4e","sender_id":1,"content":"morning! shipping the fix now","created_at":"2026-10-11T12:04:00Z"},
			{"id":"m4","channel_id":"1bbf2de4-afd9-428e-abbc-18091cb6dd4e","sender_id":3,"content":"this one is deliberately long so you can see how a message wraps inside its bubble instead of running off the edge","created_at":"2026-10-11T12:05:00Z"},
			{"id":"m5","channel_id":"1bbf2de4-afd9-428e-abbc-18091cb6dd4e","sender_id":3,"content":"and it groups with the one above","created_at":"2026-10-11T12:05:30Z"}
		]`))
	})
	return httptest.NewServer(mux)
}

func capture(t *testing.T, name string, content fyne.CanvasObject) {
	t.Helper()
	w := test.NewWindow(content)
	w.Resize(fyne.NewSize(960, 680))
	time.Sleep(400 * time.Millisecond)

	img := w.Canvas().Capture()
	var _ image.Image = img

	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", name)
	w.Close()
}

func TestRenderViews(t *testing.T) {
	srv := fakeBackend()
	defer srv.Close()

	old := baseURL
	baseURL = srv.URL
	defer func() { baseURL = old }()

	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(newTelegramTheme()) // main.go does this; the harness must too

	client := NewHTTPClient()
	client.Token = "test-token"
	client.UserID = 1 // render the "my message" (right-aligned, accent) path too

	capture(t, "shot-login.png", loginView(client, test.NewWindow(nil), func() {}, func() {}))
	capture(t, "shot-register.png", registerView(client, test.NewWindow(nil), func() {}, func() {}))

	prefs := NewPrefs(app.Preferences())
	ch := channelsView(client, prefs, test.NewWindow(nil), func(Channel) {}, func() {})
	capture(t, "shot-channels.png", ch)

	cv := chatView(client, prefs, test.NewWindow(nil), "1bbf2de4-afd9-428e-abbc-18091cb6dd4e", "general", func() {})
	capture(t, "shot-chat.png", cv)
}
