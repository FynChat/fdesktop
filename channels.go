package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// channelsView builds the channel list screen. It loads channels from the
// backend, lets the user create new channels, select one to open its chat,
// or log out. onChannelSelected is called with the selected channel on click,
// onLogout is called after the token is cleared.
func channelsView(client *HTTPClient, w fyne.Window, onChannelSelected func(channel Channel), onLogout func()) fyne.CanvasObject {
	var channels []Channel

	// Channel list: recreate/update a label per row.
	list := widget.NewList(
		func() int { return len(channels) },
		func() fyne.CanvasObject { return widget.NewLabel("channel") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(channels[i].ChannelName)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(channels) {
			onChannelSelected(channels[id])
		}
	}

	// load fetches the channel list from the backend and refreshes the list.
	// Errors are shown in a dialog.
	load := func() {
		go func() {
			chs, err := client.ListChannels()
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				channels = chs
				list.Refresh()
			})
		}()
	}
	load()

	// newChannelBtn opens a form dialog to create a channel, then reloads.
	newChannelBtn := widget.NewButton("New Channel", func() {
		typeEntry := widget.NewEntry()
		nameEntry := widget.NewEntry()
		avatarEntry := widget.NewEntry()
		dialog.ShowForm("New Channel", "Create", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Type", typeEntry),
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Avatar URL", avatarEntry),
		}, func(ok bool) {
			if !ok {
				return
			}
			go func() {
				_, err := client.CreateChannel(typeEntry.Text, nameEntry.Text, avatarEntry.Text)
				fyne.Do(func() {
					if err != nil {
						dialog.ShowError(err, w)
						return
					}
					load()
				})
			}()
		}, w)
	})

	// logoutBtn clears the auth token and returns to the login view.
	logoutBtn := widget.NewButton("Logout", func() {
		client.Token = ""
		onLogout()
	})

	return container.NewBorder(
		container.NewVBox(widget.NewLabel("Channels"), container.NewHBox(newChannelBtn, logoutBtn)),
		nil, nil, nil,
		list,
	)
}
