package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// sidebarWidth is the width of the chat list column. Telegram gives the list
// about a third of the window, which lands near this at our default size.
const sidebarWidth = 300

// channelRow is one line of the chat list: a circular avatar, the channel name
// and its type underneath, filled with the accent colour when it is the open
// chat.
//
// It holds a plain *fyne.Container rather than embedding one, because the
// canvas walks children by asserting to that exact type: a struct of our own
// that embeds one reports the right size and paints nothing.
type channelRow struct {
	container  *fyne.Container
	background *canvas.Rectangle
	avatarBox  *fyne.Container
	name       *widget.RichText
	sub        *widget.RichText
}

func newChannelRow() *channelRow {
	background := canvas.NewRectangle(color.Transparent)

	name := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyleStrong,
		Text:  "channel",
	})
	sub := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameCaptionText,
			ColorName: theme.ColorNamePlaceHolder,
		},
		Text: "group",
	})

	avatarBox := container.NewGridWrap(fyne.NewSquareSize(40), avatar("?", 40))

	// One RichText per line, not a VBox of two padded labels: padding each
	// line separately doubles the gap and makes every row far taller than the
	// chat list it is imitating.
	text := container.NewVBox(name, sub)
	body := container.NewPadded(container.NewHBox(avatarBox, text))

	return &channelRow{
		container:  container.NewStack(background, container.NewPadded(body)),
		background: background,
		avatarBox:  avatarBox,
		name:       name,
		sub:        sub,
	}
}

// update fills the row and paints its selected state. The open chat is filled
// with the accent colour and its text turns white, which is how Telegram
// shows where you are.
func (r *channelRow) update(ch Channel, selected bool) {
	r.avatarBox.Objects[0] = avatar(initial(ch.ChannelName), 40)
	r.avatarBox.Refresh()

	nameColour, subColour := theme.ColorNameForeground, theme.ColorNamePlaceHolder
	if selected {
		nameColour, subColour = theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnPrimary
		r.background.FillColor = accent
	} else {
		r.background.FillColor = color.Transparent
	}

	nameStyle := widget.RichTextStyle{
		Inline:    true,
		SizeName:  theme.SizeNameText,
		TextStyle: fyne.TextStyle{Bold: true},
		ColorName: nameColour,
	}

	r.name.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Style: nameStyle, Text: ch.ChannelName},
	}
	r.name.Refresh()

	r.sub.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Style: widget.RichTextStyle{
				SizeName:  theme.SizeNameCaptionText,
				ColorName: subColour,
			},
			Text: ch.ChannelType,
		},
	}
	r.sub.Refresh()
	r.background.Refresh()
}

// sidebarView is the left column: a search box over the chat list, with the
// new-channel, theme and log-out actions above it. Selecting a row calls
// onSelect, which swaps the content pane beside it — Telegram Desktop's
// two-pane layout, where the list stays put while you move between chats.
//
// It also returns markOpen, so the caller can highlight a chat it opened on
// its own, such as the one that was open last launch.
func sidebarView(client *HTTPClient, prefs *Prefs, w fyne.Window, onSelect func(Channel), onLogout func()) (fyne.CanvasObject, func(Channel)) {
	var (
		all     []Channel // everything the backend returned
		visible []Channel // what the search box leaves visible
		openID  string    // the channel currently shown in the content pane
	)

	banner := errorBannerWidget()

	empty := widget.NewLabel("Loading your chats…")
	empty.Alignment = fyne.TextAlignCenter
	empty.Importance = widget.LowImportance

	// rows maps the container the List holds back to the widgets inside it,
	// so the update callback never digs through nested containers.
	rows := map[*fyne.Container]*channelRow{}

	// Declared before applyFilter, which repaints it.
	var list *widget.List

	// applyFilter recomputes the visible slice from a search query. It works
	// on the data already in memory, so typing costs no requests.
	applyFilter := func(query string) {
		query = strings.ToLower(strings.TrimSpace(query))
		visible = nil
		for _, ch := range all {
			if query == "" || strings.Contains(strings.ToLower(ch.ChannelName), query) {
				visible = append(visible, ch)
			}
		}

		list.Refresh()

		empty.Hide()
		switch {
		case len(visible) > 0:
			list.Show()
		case len(all) == 0:
			empty.SetText("No channels yet.\nCreate one with the + button above.")
			empty.Show()
		default:
			empty.SetText("No channel matches that.")
			empty.Show()
		}
	}

	list = widget.NewList(
		func() int { return len(visible) },
		func() fyne.CanvasObject {
			row := newChannelRow()
			rows[row.container] = row
			return row.container
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			row, ok := rows[o.(*fyne.Container)]
			if !ok || int(i) >= len(visible) {
				return
			}
			row.update(visible[i], visible[i].ID == openID)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || int(id) >= len(visible) {
			return
		}
		openID = visible[id].ID
		list.Refresh()
		onSelect(visible[id])
	}

	// markOpen highlights a chat opened from outside the list, which happens
	// when the app reopens the conversation from last time.
	markOpen := func(ch Channel) {
		openID = ch.ID
		list.Refresh()
	}

	// A plain entry: Fyne has no dedicated search widget, and the placeholder
	// carries the meaning. Typing filters the list without a request.
	search := widget.NewEntry()
	search.SetPlaceHolder("Search channels")
	search.OnChanged = applyFilter

	// load fetches the channel list from the backend.
	load := func() {
		go func() {
			chs, err := client.ListChannels()

			fyne.Do(func() {
				if err != nil {
					banner.Show(err)
					return
				}
				banner.Clear()
				all = chs
				applyFilter(search.Text)
			})
		}()
	}
	load()

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

	themeBtn := iconButton(theme.ColorPaletteIcon(), func() {
		if prefs == nil {
			return
		}
		prefs.SetDarkTheme(!prefs.DarkTheme())
		prefs.apply()
	})

	logoutBtn := iconButton(theme.DeleteIcon(), func() {
		client.Logout()
		onLogout()
	})

	header := container.NewPadded(container.NewBorder(nil, nil, newChannelBtn,
		container.NewHBox(themeBtn, logoutBtn), search))

	body := container.NewStack(container.NewCenter(empty), container.NewPadded(list))

	// A Border layout, not a padded box: the list has to fill the pane and
	// take all the space the header leaves, however tall the window is.
	return container.NewBorder(header, banner.wrap, nil, nil, body), markOpen
}
