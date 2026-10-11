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
	w := a.NewWindow("FynChat")

	client := NewHTTPClient()
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

	showLogin()
	w.Resize(fyne.NewSize(420, 560))
	w.ShowAndRun()
}
