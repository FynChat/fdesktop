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
	badge      *fyne.Container
	name       *widget.RichText
}

func newChannelRow() *channelRow {
	background := canvas.NewRectangle(color.Transparent)

	name := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyleStrong,
		Text:  "channel",
	})

	// A small badge on the right says what kind of channel this is. A
	// one-to-one conversation has none: the person's avatar already says it.
	badge := container.NewCenter(container.NewGridWrap(fyne.NewSquareSize(16),
		canvas.NewRectangle(color.Transparent)))

	avatarBox := container.NewGridWrap(fyne.NewSquareSize(40), avatar("?", 40))

	// One RichText, not a VBox of two lines: the row is avatar, name and
	// badge, and the type is no longer spelled out underneath it.
	body := container.NewBorder(nil, nil, avatarBox, badge, container.NewPadded(name))

	return &channelRow{
		container:  container.NewStack(background, body),
		background: background,
		avatarBox:  avatarBox,
		name:       name,
		badge:      badge,
	}
}

// update fills the row, paints its selected state and swaps in the badge for
// the channel's kind.
func (r *channelRow) update(ch Channel, selected bool) {
	r.avatarBox.Objects[0] = avatar(initial(ch.ChannelName), 40)
	r.avatarBox.Refresh()

	if icon := kindIcon(ch.ChannelType); icon != nil {
		r.badge.Objects[0] = container.NewGridWrap(fyne.NewSquareSize(16), icon)
		r.badge.Show()
	} else {
		r.badge.Hide()
	}
	r.badge.Refresh()

	// The selected row flips to white text on the accent blue, which is how
	// Telegram shows the open chat.
	nameColour := theme.ColorNameForeground
	if selected {
		nameColour = theme.ColorNameForegroundOnPrimary
		r.background.FillColor = accent
	} else {
		r.background.FillColor = color.Transparent
	}

	r.name.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Style: widget.RichTextStyle{
				Inline:    true,
				SizeName:  theme.SizeNameText,
				TextStyle: fyne.TextStyle{Bold: true},
				ColorName: nameColour,
			},
			Text: ch.ChannelName,
		},
	}
	r.name.Refresh()
	r.background.Refresh()
}

// sidebar is the left column plus the two things the shell needs to wire it
// into the app: a way to highlight a row it opened by itself, and the search
// entry, which now lives in the side menu rather than the header.
type sidebar struct {
	view     fyne.CanvasObject
	markOpen func(Channel)
	search   *widget.Entry
}

// sidebarView is the left column: the app name with the menu button in the
// corner, over the chat list. Selecting a row calls onSelect, which swaps the
// content pane beside it — Telegram Desktop's two-pane layout, where the list
// stays put while you move between chats.
func sidebarView(client *HTTPClient, prefs *Prefs, w fyne.Window, onSelect func(Channel), onLogout, onMenu func()) sidebar {
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

	// The header is the app name and one button. Everything else — new
	// channel, theme, log out, search — lives in the menu that slides out
	// from it, which is where Telegram keeps them too.
	appName := canvas.NewText("FynChat", theme.ForegroundColor())
	appName.TextStyle = fyne.TextStyle{Bold: true}
	appName.TextSize = theme.Size(theme.SizeNameSubHeadingText)

	header := container.NewPadded(container.NewBorder(nil, nil, container.NewPadded(appName),
		container.NewHBox(newChannelBtn, iconButton(theme.MenuIcon(), onMenu)), nil))

	body := container.NewStack(container.NewCenter(empty), container.NewPadded(list))

	// A Border layout, not a padded box: the list has to fill the pane and
	// take all the space the header leaves, however tall the window is.
	view := container.NewBorder(header, banner.wrap, nil, nil, body)

	return sidebar{view: view, markOpen: markOpen, search: search}
}
