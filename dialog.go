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

// Only the kinds of channel that are made by naming them.
//
// A private chat is deliberately absent. It is not created from a form: it is
// what you get by choosing a person from your contacts, and the backend decides
// whether that means making a channel or opening the one that already exists.
// That is the same split Telegram makes — "New group" and "New channel" ask for
// a name, and people come from Contacts.
var channelTypes = []channelType{
	{"group", "Group", "Many people, anyone with the link can join"},
	{"channel", "Channel", "Broadcast: everyone reads, only owners post"},
}

// cleanChannelInput trims what was typed and rejects the one thing the client
// can catch on its own: a named channel with no name. Kept separate from the
// dialog so the rules can be tested without building a window.
func cleanChannelInput(kind, name, avatar string) (string, string, string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	name = strings.TrimSpace(name)
	avatar = strings.TrimSpace(avatar)

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

	if name == "" {
		return "", "", "", errors.New("Give your channel a name")
	}

	return kind, name, avatar, nil
}

// cleanContactHandle normalises what the user typed into the handle the backend
// expects.
//
// The leading @ is the part worth handling: people type it out of habit, and
// "＠" and "@" are different characters to a database. Trimming it here means
// the backend only ever sees one spelling of a handle.
func cleanContactHandle(raw string) (string, error) {
	handle := strings.TrimSpace(raw)
	handle = strings.TrimPrefix(handle, "@")

	if handle == "" {
		return "", errors.New("Enter the username of the person to add")
	}
	if strings.ContainsAny(handle, " \t") {
		return "", errors.New("A username has no spaces in it")
	}

	return handle, nil
}

// newChannelDialog asks for the details of a new channel. The type is picked
// from a small set of buttons rather than typed into a box: the backend only
// accepts these values, and a typo there comes back as a 400 nobody enjoys.
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

// addContactDialog asks for the username of somebody to add to the contacts.
//
// It asks for a handle rather than a display name because a handle is what
// names an account: typing "Ada" cannot be resolved to a person, and letting
// the user pick the name is how one person ends up as two contacts. The name
// comes back from the backend, from the account that was found.
//
// onAdd is called with the handle; returning an error keeps the dialog open and
// shows the message, so "no such user" is read where it was typed.
func addContactDialog(parent fyne.Window, onAdd func(username string) error) {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("@ada")
	problem := widget.NewLabel("")
	problem.Importance = widget.LowImportance
	problem.Hide()

	content := container.NewVBox(
		widget.NewLabel("Enter their username. You can start a chat as soon as they are in your contacts."),
		usernameEntry,
		problem,
	)

	// Enter is the natural way to submit one field, so it is wired to the same
	// path the button takes.
	var d *dialog.ConfirmDialog
	submit := func() {
		handle, err := cleanContactHandle(usernameEntry.Text)
		if err != nil {
			problem.SetText(err.Error())
			problem.Show()
			return
		}

		go func() {
			err := onAdd(handle)

			fyne.Do(func() {
				if err != nil {
					problem.SetText(err.Error())
					problem.Show()
					return
				}
				d.Hide()
			})
		}()
	}
	usernameEntry.OnSubmitted = func(string) { submit() }

	d = dialog.NewCustomConfirm("Add contact", "Add", "Cancel", container.NewPadded(content),
		func(confirmed bool) {
			if confirmed {
				submit()
			}
		}, parent)

	d.Show()
}
