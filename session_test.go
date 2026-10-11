package main

import (
	"net/http"
	"testing"

	"fyne.io/fyne/v2/test"
)

// The session must survive a restart: login stores it, a fresh client reads
// it back, and Logout clears it again.
func TestSessionPersistence(t *testing.T) {
	newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"token":"jwt-xyz","user":{"id":7,"name":"Ada"}}`))
	})

	app := test.NewApp()
	defer app.Quit()

	// First run: log in.
	first := NewHTTPClient()
	first.AttachPersistence(app.Preferences())
	if _, err := first.Login("ada@example.com", "hunter22"); err != nil {
		t.Fatal(err)
	}
	if first.Token != "jwt-xyz" || first.UserID != 7 {
		t.Fatalf("after login: token=%q user=%d", first.Token, first.UserID)
	}

	// Second run: a brand new client restores the session from disk.
	second := NewHTTPClient()
	second.AttachPersistence(app.Preferences())
	if second.Token != "jwt-xyz" {
		t.Errorf("token not persisted: %q", second.Token)
	}
	if second.UserID != 7 {
		t.Errorf("user id not persisted: %d", second.UserID)
	}

	// Logging out must clear both, in memory and on disk.
	second.Logout()
	if second.Token != "" || second.UserID != 0 {
		t.Errorf("after logout: token=%q user=%d", second.Token, second.UserID)
	}

	third := NewHTTPClient()
	third.AttachPersistence(app.Preferences())
	if third.Token != "" {
		t.Errorf("logout not persisted: %q", third.Token)
	}
}
