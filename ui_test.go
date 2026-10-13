package main

import (
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

// The side-menu scrim used to be an invisible widget.Button stacked over the
// whole window. A Stack hands every child the full size, so that button sat on
// top of the sidebar and the chat and swallowed every tap — the list looked
// fine but nothing in it could be clicked. The scrim must instead start hidden
// and only appear while the menu is open.
func TestScrimStartsHiddenAndClosesOnTap(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(newTelegramTheme())

	var taps int32
	s := newScrimWidget(func() { atomic.AddInt32(&taps, 1) })

	if s.Visible() {
		t.Fatal("scrim must start hidden so it cannot block taps on the sidebar")
	}

	// Shown only while the menu is open, hidden again once it closes.
	s.Show()
	if !s.Visible() {
		t.Error("scrim must become visible when the menu opens")
	}

	// A tap on the visible scrim runs the close handler exactly once.
	s.Tapped(nil)
	if got := atomic.LoadInt32(&taps); got != 1 {
		t.Errorf("one tap ran the handler %d times, want 1", got)
	}

	s.Hide()
	if s.Visible() {
		t.Error("scrim must be hidden again after the menu closes")
	}
}

// The menu's own scrim is wired through buildScrim: a fresh sideMenu must not
// expose a scrim that covers the window, and toggling the menu must show then
// hide it again.
func TestSideMenuScrimFollowsOpenState(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(newTelegramTheme())

	m := &sideMenu{panel: container.NewStack()}
	m.buildScrim()

	if m.scrim == nil {
		t.Fatal("buildScrim must populate the scrim")
	}
	if m.scrim.Visible() {
		t.Error("a closed menu must not show its scrim")
	}

	m.toggle(fyne.NewSize(800, 600))
	if !m.open {
		t.Fatal("toggle must open the menu")
	}
	if !m.scrim.Visible() {
		t.Error("an open menu must show its scrim")
	}

	m.toggle(fyne.NewSize(800, 600))
	if m.open {
		t.Fatal("toggle must close the menu")
	}
	// The scrim is hidden after the slide finishes (a ~200 ms timer), so wait.
	if !waitFor(func() bool { return !m.scrim.Visible() }) {
		t.Error("a closed menu must hide its scrim once the slide completes")
	}
}
