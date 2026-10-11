package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// channelRow holds the widgets of one list line: the container the List keeps,
// plus the pieces inside it that update() has to reach.
type channelRow struct {
	container *fyne.Container
	avatarBox *fyne.Container
	labels    *widget.RichText
}

func newChannelRow() *channelRow {
	// One RichText instead of two labels: a VBox stretches its children to
	// fill the row height, which pushes the second line far below the first.
	// A single object keeps the two lines tight.
	labels := widget.NewRichText()

	avatarBox := container.NewGridWrap(fyne.NewSquareSize(36), avatar("?", 36))

	return &channelRow{
		container: container.NewPadded(container.NewHBox(avatarBox, labels)),
		avatarBox: avatarBox,
		labels:    labels,
	}
}

// update fills the row with one channel's data.
func (r *channelRow) update(ch Channel) {
	r.avatarBox.Objects[0] = avatar(ch.ChannelName, 36)
	r.avatarBox.Refresh()

	r.labels.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  ch.ChannelName,
		},
		&widget.TextSegment{
			Style: widget.RichTextStyle{
				SizeName:  theme.SizeNameCaptionText,
				ColorName: theme.ColorNamePlaceHolder,
				TextStyle: fyne.TextStyle{Italic: true},
			},
			Text: "\n" + ch.ChannelType,
		},
	}
	r.labels.Refresh()
}

// channelsView builds the channel list screen. It loads channels from the
// backend, lets the user create new channels, select one to open its chat,
// or log out. onChannelSelected is called with the selected channel on click,
// onLogout is called after the token is cleared.
func channelsView(client *HTTPClient, prefs *Prefs, w fyne.Window, onChannelSelected func(channel Channel), onLogout func()) fyne.CanvasObject {
	var channels []Channel
	banner := errorBannerWidget()

	// The row stays a plain *fyne.Container on purpose: the canvas walks
	// children by type-asserting to that exact type, so a struct of our own
	// that embeds one reports the right size but paints nothing. The widgets
	// are therefore looked up by pointer instead of through nesting.
	rows := map[*fyne.Container]*channelRow{}

	// Empty state, shown while the first load is in flight or when the user
	// has no channels yet.
	empty := widget.NewLabel("Loading your chats…")
	empty.Alignment = fyne.TextAlignCenter
	empty.Importance = widget.LowImportance

	list := widget.NewList(
		func() int { return len(channels) },
		func() fyne.CanvasObject {
			row := newChannelRow()
			rows[row.container] = row
			return row.container
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			row, ok := rows[o.(*fyne.Container)]
			if !ok {
				return
			}
			row.update(channels[i])
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(channels) {
			onChannelSelected(channels[id])
		}
	}

	// load fetches the channel list from the backend and refreshes the list.
	load := func() {
		go func() {
			chs, err := client.ListChannels()

			fyne.Do(func() {
				if err != nil {
					banner.Show(err)
					return
				}
				banner.Clear()
				channels = chs
				list.Refresh()
				empty.Hide()
				if len(channels) > 0 {
					list.Show()
				}
			})
		}()
	}
	load()

	// logoutBtn clears the auth token and returns to the login view.
	logoutBtn := iconButton(theme.DeleteIcon(), func() {
		client.Logout()
		onLogout()
	})

	themeBtn := iconButton(theme.ColorPaletteIcon(), func() {
		if prefs == nil {
			return
		}
		prefs.SetDarkTheme(!prefs.DarkTheme())
		prefs.apply()
	})

	refreshBtn := iconButton(theme.ViewRefreshIcon(), func() {
		banner.Clear()
		load()
	})

	// The header actions are icons, not labelled buttons: the header is
	// already a title, and icons keep the row short at any window width.
	newChannelBtn := iconButton(theme.ContentAddIcon(), func() {
		newChannelDialog(w, func(kind, name, avatar string) error {
			_, err := client.CreateChannel(kind, name, avatar)
			if err != nil {
				return err
			}

			fyne.Do(load)
			return nil
		})
	})

	header := headerBar("Chats", newChannelBtn, themeBtn, refreshBtn, logoutBtn)
	body := container.NewStack(container.NewCenter(empty), container.NewPadded(list))

	return container.NewBorder(header, banner.wrap, nil, nil, body)
}
