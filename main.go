package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

// main creates the Fyne app, a single window, and the shared HTTPClient,
// then wires the views together:
//
//	Login ⇄ Register → Channels → Chat
//	Chat --Back--> Channels --Logout--> Login
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

	var selectedChannel Channel

	var showLogin, showRegister, showChannels, showChat func()

	showLogin = func() {
		w.SetContent(loginView(client, w, showChannels, showRegister))
	}
	showRegister = func() {
		w.SetContent(registerView(client, w, showChannels, showLogin))
	}
	showChannels = func() {
		w.SetContent(channelsView(client, prefs, w, func(ch Channel) {
			selectedChannel = ch
			showChat()
		}, showLogin))
	}
	showChat = func() {
		w.SetContent(chatView(client, prefs, w,
			selectedChannel.ID, selectedChannel.ChannelName, showChannels))
	}

	// An expired token would otherwise fail every request with a dialog. The
	// client tells us once, and we drop back to the login screen.
	client.SetSessionExpiredHandler(func() {
		fyne.Do(func() {
			client.Logout()
			showLogin()
		})
	})

	// A remembered token means we can skip the login screen. If it turned out
	// to be expired, the channels view shows the error and its Log out button
	// brings the user back here.
	if client.Token == "" {
		showLogin()
	} else {
		showChannels()

		// Reopen the channel that was open last time, if it is still there.
		if last := prefs.LastChannel(); last != "" {
			go func() {
				channels, err := client.ListChannels()

				fyne.Do(func() {
					if err != nil {
						return
					}
					for _, ch := range channels {
						if ch.ID == last {
							selectedChannel = ch
							showChat()
							return
						}
					}
				})
			}()
		}
	}

	// Big enough to read a conversation without scrolling sideways, small
	// enough to sit next to a terminal. The window stays resizable.
	w.Resize(fyne.NewSize(960, 680))
	w.ShowAndRun()
}
