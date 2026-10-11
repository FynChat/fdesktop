package main

import (
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// asyncButton guards the login and register forms, so it has to actually run
// when tapped. Its first version had the guard inverted and the button did
// nothing at all: the app looked fine and only a real click revealed it.
//
// The assertions stop at the point where the background action is released,
// because Fyne's test driver runs fyne.Do inline on the calling goroutine:
// reading the button after the release would race with the widget update. The
// busy-to-idle transition is therefore checked by hand, not here.
func TestAsyncButtonRunsAndLocks(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(newTelegramTheme())

	var calls int32
	release := make(chan struct{})

	b := asyncButton("Log In", "Signing in…", func() {
		atomic.AddInt32(&calls, 1)
		<-release // hold the button in its busy state
	})

	w := test.NewWindow(b)
	defer w.Close()
	w.Resize(fyne.NewSize(200, 80))

	if b.Disabled() {
		t.Fatal("a fresh button must start enabled")
	}
	if b.Text != "Log In" {
		t.Errorf("initial text = %q, want %q", b.Text, "Log In")
	}

	// A tap on an enabled button must run the action. It runs on a goroutine,
	// so wait for it rather than checking immediately.
	b.OnTapped()
	if !waitFor(func() bool { return atomic.LoadInt32(&calls) == 1 }) {
		t.Fatalf("action ran %d times after one tap, want 1", atomic.LoadInt32(&calls))
	}

	// The action is blocked, so nothing else is touching the button and these
	// reads are safe.
	if !b.Disabled() {
		t.Error("button must be disabled while the action runs")
	}
	if b.Text != "Signing in…" {
		t.Errorf("text while busy = %q, want %q", b.Text, "Signing in…")
	}

	// A second tap while the action is in flight must be ignored.
	b.OnTapped()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("a busy button ran the action again (%d calls)", got)
	}

	close(release)
	time.Sleep(100 * time.Millisecond) // let the goroutine finish the update
}

func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}
