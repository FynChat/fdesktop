package main

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// channelType describes one of the kinds of channel the backend accepts, and
// the words to use for it in the picker.
type channelType struct {
	value string
	label string
	blurb string
}

var channelTypes = []channelType{
	{"group", "Group", "Many people, anyone with the link can join"},
	{"channel", "Channel", "Broadcast: everyone reads, only owners post"},
	{"p2p", "Private", "One other person, just the two of you"},
}

// cleanChannelInput trims what was typed and rejects the one thing the client
// can catch on its own: a channel with no name. Kept separate from the dialog
// so the rules can be tested without building a window.
func cleanChannelInput(kind, name, avatar string) (string, string, string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	name = strings.TrimSpace(name)
	avatar = strings.TrimSpace(avatar)

	if name == "" {
		return "", "", "", errors.New("Give your channel a name")
	}

	known := false
	for _, ct := range channelTypes {
		if ct.value == kind {
			known = true
			break
		}
	}
	if !known {
		return "", "", "", fmt.Errorf("pick one of the channel types shown")
	}

	return kind, name, avatar, nil
}

// newChannelDialog asks for the details of a new channel. The type is picked
// from a small set of buttons rather than typed into a box: the backend only
// accepts three values, and a typo there comes back as a 400 nobody enjoys.
//
// onCreate is called with the chosen values; returning an error keeps the
// dialog open and shows the message, which is how a name clash or a rejected
// type is reported without the dialog vanishing first.
func newChannelDialog(parent fyne.Window, onCreate func(kind, name, avatar string) error) {
	selected := channelTypes[0]
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("general")
	avatarEntry := widget.NewEntry()
	avatarEntry.SetPlaceHolder("https://… (optional)")
	problem := widget.NewLabel("")
	problem.Importance = widget.LowImportance
	problem.Hide()

	// The avatar preview updates as the name is typed, so the row shows what
	// the channel will look like in the list.
	preview := container.NewGridWrap(fyne.NewSquareSize(48), avatar(selected.label, 48))
	nameEntry.OnChanged = func(text string) {
		label := strings.TrimSpace(text)
		if label == "" {
			label = selected.label
		}
		preview.Objects[0] = avatar(label, 48)
		preview.Refresh()
	}

	// Type picker: one button per kind, the chosen one filled in.
	var typeButtons []*widget.Button
	typeRow := container.NewHBox()
	blurb := widget.NewLabel(channelTypes[0].blurb)
	blurb.Importance = widget.LowImportance

	refreshTypes := func() {
		for i, b := range typeButtons {
			if channelTypes[i].value == selected.value {
				b.Importance = widget.HighImportance
			} else {
				b.Importance = widget.LowImportance
			}
			b.Refresh()
		}
		blurb.SetText(selected.blurb)
	}

	for _, ct := range channelTypes {
		ct := ct
		btn := widget.NewButton(ct.label, func() {
			selected = ct
			refreshTypes()
		})
		typeButtons = append(typeButtons, btn)
		typeRow.Add(btn)
	}
	refreshTypes()

	content := container.NewVBox(
		widget.NewLabelWithStyle("What kind of channel?", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		typeRow,
		blurb,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nameEntry,
		widget.NewLabelWithStyle("Avatar", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		avatarEntry,
		container.NewHBox(preview, widget.NewLabel("Shown in the channel list")),
		problem,
	)

	// Declared first: the confirm handler needs to close the dialog once the
	// backend accepts, but it is also the argument that creates it.
	var d *dialog.ConfirmDialog
	d = dialog.NewCustomConfirm("New channel", "Create", "Cancel", container.NewPadded(content),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			kind, name, avatar, err := cleanChannelInput(selected.value, nameEntry.Text, avatarEntry.Text)
			if err != nil {
				problem.SetText(err.Error())
				problem.Show()
				return
			}

			go func() {
				err := onCreate(kind, name, avatar)

				fyne.Do(func() {
					if err != nil {
						problem.SetText(err.Error())
						problem.Show()
						return
					}
					d.Hide()
				})
			}()
		}, parent)

	d.Show()
}
