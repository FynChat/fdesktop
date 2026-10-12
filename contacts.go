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

// contactsView is the Contacts screen: the people the user can start a private
// chat with, and the button that adds one.
//
// It is a screen of its own rather than a section of the chat list because it
// answers a different question. The chat list is "what am I in the middle of",
// contacts is "who exists", and the two have different actions: one opens a
// conversation that is already there, the other may have to make one.
//
// Tapping a contact calls onOpen with the channel for that person. The backend
// decides whether that meant creating the chat or handing back the one that
// already exists, so this view does not have to know, and cannot get it wrong.
func contactsView(client *HTTPClient, w fyne.Window, onOpen func(Channel), onBack func()) fyne.CanvasObject {
	var (
		all     []Contact // everything the backend returned
		visible []Contact // what the search box leaves visible
	)

	banner := errorBannerWidget()

	empty := widget.NewLabel("Loading your contacts…")
	empty.Alignment = fyne.TextAlignCenter
	empty.Importance = widget.LowImportance

	// rows maps the container the List holds back to the widgets inside it, so
	// the update callback never digs through nested containers.
	rows := map[*fyne.Container]*contactRow{}

	// Declared before applyFilter, which repaints it.
	var list *widget.List

	// applyFilter recomputes the visible slice from a search query, working on
	// what is already in memory so typing costs no requests.
	applyFilter := func(query string) {
		visible = contactFilter(all, query)
		list.Refresh()

		empty.Hide()
		switch {
		case len(visible) > 0:
			list.Show()
		case len(all) == 0:
			empty.SetText("No contacts yet.\nAdd someone with the + button above.")
			empty.Show()
		default:
			empty.SetText("No contact matches that.")
			empty.Show()
		}
	}

	list = widget.NewList(
		func() int { return len(visible) },
		func() fyne.CanvasObject {
			row := newContactRow()
			rows[row.container] = row
			return row.container
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			row, ok := rows[o.(*fyne.Container)]
			if !ok || int(i) >= len(visible) {
				return
			}
			row.update(visible[i])
		},
	)

	// Opening a chat is a request, so the row that was tapped is marked busy
	// until it comes back. Without it a slow connection looks like a tap that
	// did nothing, and gets tapped again.
	opening := false
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || int(id) >= len(visible) || opening {
			return
		}
		contact := visible[id]
		opening = true
		list.UnselectAll()

		go func() {
			channel, err := client.OpenPrivateChat(contact.ID)

			fyne.Do(func() {
				opening = false
				if err != nil {
					banner.Show(err)
					return
				}
				banner.Clear()
				onOpen(channel)
			})
		}()
	}

	search := widget.NewEntry()
	search.SetPlaceHolder("Search contacts")
	search.OnChanged = applyFilter

	// load fetches the contact list from the backend.
	load := func() {
		go func() {
			contacts, err := client.ListContacts()

			fyne.Do(func() {
				if err != nil {
					banner.Show(err)
					return
				}
				banner.Clear()
				all = contacts
				applyFilter(search.Text)
			})
		}()
	}
	load()

	addBtn := iconButton(theme.ContentAddIcon(), func() {
		addContactDialog(w, func(username string) error {
			contact, err := client.AddContact(username)
			if err != nil {
				return err
			}

			// The new contact is prepended rather than the whole list being
			// refetched: the person just added is the one the user is looking
			// for, and a round trip would only make them wait to see it.
			fyne.Do(func() {
				all = append([]Contact{contact}, all...)
				applyFilter(search.Text)
			})
			return nil
		})
	})

	title := canvas.NewText("Contacts", theme.ForegroundColor())
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = theme.Size(theme.SizeNameSubHeadingText)

	header := container.NewPadded(container.NewBorder(nil, nil,
		container.NewHBox(iconButton(theme.NavigateBackIcon(), onBack), container.NewPadded(title)),
		addBtn, nil))

	body := container.NewStack(container.NewCenter(empty), container.NewPadded(list))

	// The search entry goes under the header, in the top region of the Border.
	// Putting it in the left slot (as this first did) stretches it down the
	// whole window instead of across, because that slot is for a side panel.
	view := container.NewBorder(
		container.NewVBox(header, container.NewPadded(search)),
		banner.wrap,
		nil, nil,
		body,
	)

	return view
}

// contactRow is one line of the contact list: a circular avatar, the person's
// name and their handle underneath.
//
// It holds a plain *fyne.Container rather than embedding one, because the canvas
// walks children by asserting to that exact type: a struct of our own that
// embeds one reports the right size and paints nothing.
type contactRow struct {
	container *fyne.Container
	avatarBox *fyne.Container
	name      *widget.RichText
	handle    *widget.RichText
}

func newContactRow() *contactRow {
	name := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyleStrong,
		Text:  "name",
	})

	handle := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			Inline:    true,
			SizeName:  theme.SizeNameCaptionText,
			ColorName: theme.ColorNamePlaceHolder,
		},
		Text: "@handle",
	})

	avatarBox := container.NewGridWrap(fyne.NewSquareSize(40), avatar("?", 40))

	body := container.NewBorder(nil, nil,
		avatarBox,
		nil,
		container.NewPadded(container.NewVBox(name, handle)),
	)

	return &contactRow{
		container: container.NewStack(
			canvas.NewRectangle(color.Transparent),
			body,
		),
		avatarBox: avatarBox,
		name:      name,
		handle:    handle,
	}
}

// update fills the row, including hiding the handle when the person has none:
// an "@" with nothing after it reads as a bug.
func (r *contactRow) update(c Contact) {
	r.avatarBox.Objects[0] = avatar(initial(c.Name), 40)
	r.avatarBox.Refresh()

	r.name.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Style: widget.RichTextStyle{
				Inline:    true,
				SizeName:  theme.SizeNameText,
				TextStyle: fyne.TextStyle{Bold: true},
				ColorName: theme.ColorNameForeground,
			},
			Text: c.Name,
		},
	}
	r.name.Refresh()

	if c.Username == "" {
		r.handle.Hide()
	} else {
		r.handle.Segments = []widget.RichTextSegment{
			&widget.TextSegment{
				Style: widget.RichTextStyle{
					Inline:    true,
					SizeName:  theme.SizeNameCaptionText,
					ColorName: theme.ColorNamePlaceHolder,
				},
				Text: "@" + c.Username,
			},
		}
		r.handle.Show()
		r.handle.Refresh()
	}
}

// contactFilter narrows a contact list by a search string, matching on both the
// name and the handle. An empty search keeps everything, so the list is never
// mysteriously blank.
func contactFilter(contacts []Contact, query string) []Contact {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return contacts
	}

	matched := make([]Contact, 0, len(contacts))
	for _, c := range contacts {
		if strings.Contains(strings.ToLower(c.Name), query) ||
			strings.Contains(strings.ToLower(c.Username), query) {
			matched = append(matched, c)
		}
	}
	return matched
}
