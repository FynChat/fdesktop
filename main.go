package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	// NewWithID, not New: the Preferences API refuses to store anything
	// without a unique app ID, which would silently break session persistence.
	a := app.NewWithID("com.fynchat.desktop")

	prefs := NewPrefs(a.Preferences())
	prefs.apply() // restore the theme chosen last time

	w := a.NewWindow("FynChat")
	w.SetIcon(appIcon())

	client := NewHTTPClient()
	client.AttachPersistence(a.Preferences())

	// Demo mode serves canned data from memory and signs straight in, so the
	// layout can be looked at without a backend.
	demo := demoEnabled()
	if demo {
		url, stop := startDemoServer()
		defer stop()
		baseURL = url

		if _, err := client.Login("demo@example.com", "demo"); err != nil {
			log.Fatalf("demo: %v", err)
		}
	}

	// Declared before the handlers below, which close over them.
	var showLogin, showRegister func()

	// home swaps the whole window for the two-pane shell and returns the
	// function that opens a chat in its right-hand pane.
	var home func() func(Channel)
	home = func() func(Channel) {
		layout, openChat := shell(client, prefs, w, showLogin)
		w.SetContent(layout)
		return openChat
	}

	showLogin = func() {
		w.SetContent(loginView(client, w, func() { home() }, showRegister))
	}
	showRegister = func() {
		w.SetContent(registerView(client, w, func() { home() }, showLogin))
	}

	// An expired token would otherwise fail every request with a dialog. The
	// client tells us once, and we drop back to the login screen.
	client.SetSessionExpiredHandler(func() {
		fyne.Do(func() {
			client.Logout()
			showLogin()
		})
	})

	if demo || client.Token != "" {
		openChat := home()

		// Reopen the channel that was open last time, if it still exists.
		if last := prefs.LastChannel(); last != "" {
			go func() {
				channels, err := client.ListChannels()
				if err != nil {
					return
				}

				fyne.Do(func() {
					for _, ch := range channels {
						if ch.ID == last {
							openChat(ch)
							return
						}
					}
				})
			}()
		}
	} else {
		showLogin()
	}

	// Two panes need room: the list is a fixed 300 points wide.
	w.Resize(fyne.NewSize(1000, 680))
	w.ShowAndRun()
}
