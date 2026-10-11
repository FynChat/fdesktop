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
	a := app.New()
	a.Settings().SetTheme(newTelegramTheme())

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
		w.SetContent(channelsView(client, w, func(ch Channel) {
			selectedChannel = ch
			showChat()
		}, showLogin))
	}
	showChat = func() {
		w.SetContent(chatView(client, w, selectedChannel.ID, selectedChannel.ChannelName, showChannels))
	}

	// A remembered token means we can skip the login screen. If it turned out
	// to be expired, the channels view shows the error and its Log out button
	// brings the user back here.
	if client.Token != "" {
		showChannels()
	} else {
		showLogin()
	}
	// Big enough to read a conversation without scrolling sideways, small
	// enough to sit next to a terminal. The window stays resizable.
	w.Resize(fyne.NewSize(960, 680))
	w.ShowAndRun()
}
