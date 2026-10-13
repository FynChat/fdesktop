package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// settingsView is the Settings screen: who you are, how the app looks, and how
// to get out. It is a screen of its own rather than a menu because these are
// answers to "how is this set up", which rewards seeing them all at once with
// room to explain, instead of one tap-toggle per row.
//
// It swaps into the left pane the same way Contacts does, so the chat list
// stays put and the conversation beside it does not change.
func settingsView(client *HTTPClient, prefs *Prefs, w fyne.Window, onBack, onLogout func()) fyne.CanvasObject {
	name := client.Name
	if name == "" {
		name = "Your account"
	}

	// Account card: the avatar, the name and the id the backend knows them by.
	userName := canvas.NewText(name, theme.ForegroundColor())
	userName.TextStyle = fyne.TextStyle{Bold: true}
	userName.TextSize = theme.Size(theme.SizeNameSubHeadingText)

	userSub := canvas.NewText("User ID: "+fmt.Sprintf("%d", client.UserID), theme.PlaceHolderColor())
	userSub.TextSize = theme.Size(theme.SizeNameCaptionText)

	account := settingsCard(container.NewHBox(
		container.NewGridWrap(fyne.NewSquareSize(52), avatar(initial(name), 52)),
		container.NewPadded(container.NewVBox(userName, userSub)),
	))

	// Appearance: one switch rather than a menu item that hides its own state.
	// The note beside the menu's Theme row said "Dark"/"Light"; a labelled
	// switch says which is on without a second glance.
	themeSwitch := widget.NewCheck("", func(on bool) {
		if prefs == nil {
			return
		}
		prefs.SetDarkTheme(on)
		prefs.apply()
	})
	if prefs != nil {
		themeSwitch.SetChecked(prefs.DarkTheme())
	}
	themeRow := settingsRow("Dark theme", "Applies across the whole app", themeSwitch)

	// Backend: read-only, because the URL comes from FYNCHAT_API_URL and an
	// in-app field that does nothing would be worse than none at all.
	backend := settingsCard(container.NewVBox(
		settingsRow("Server", baseURL, nil),
	))

	// Shortcuts: the list the app actually binds, plus the button that shows
	// the dialog. Duplicating the keys here would let them drift, so the help
	// dialog is the single source and this is the way to reach it.
	shortcutsBtn := widget.NewButtonWithIcon("View all", theme.ListIcon(), func() {
		showShortcutsDialog(w)
	})
	shortcutsBtn.Importance = widget.LowImportance
	shortcutsCard := settingsCard(container.NewVBox(
		settingsRow("Keyboard shortcuts", "Ctrl + Enter sends, Ctrl + N makes a channel", shortcutsBtn),
	))

	logoutBtn := widget.NewButtonWithIcon("Log out", theme.LogoutIcon(), onLogout)
	logoutBtn.Importance = widget.DangerImportance

	title := canvas.NewText("Settings", theme.ForegroundColor())
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = theme.Size(theme.SizeNameSubHeadingText)

	header := container.NewPadded(container.NewBorder(nil, nil,
		container.NewHBox(iconButton(theme.NavigateBackIcon(), onBack), container.NewPadded(title)),
		nil, nil))

	// The cards scroll: four sections and a button do not fit a short window,
	// and pushing the log-out button off the bottom is how it gets missed.
	body := container.NewVBox(
		sectionLabel("Account"),
		account,
		gap(8),
		sectionLabel("Appearance"),
		themeRow,
		gap(8),
		sectionLabel("Server"),
		backend,
		gap(8),
		shortcutsCard,
		gap(8),
		logoutBtn,
		gap(16),
	)

	return container.NewBorder(header, nil, nil, nil,
		container.NewVScroll(container.NewPadded(body)))
}

// settingsCard is one raised block of settings, the same surface the auth
// screens use, so a group of rows reads as one thing.
func settingsCard(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(cardBackground())
	bg.CornerRadius = 12
	return container.NewStack(bg, container.NewPadded(content))
}

// settingsRow is a label with an optional note underneath and a control on the
// right. A nil control makes it a read-only row, which is how the server URL
// is shown.
func settingsRow(label, note string, control fyne.CanvasObject) fyne.CanvasObject {
	head := canvas.NewText(label, theme.ForegroundColor())
	head.TextStyle = fyne.TextStyle{Bold: true}

	var objects []fyne.CanvasObject
	if note != "" {
		sub := canvas.NewText(note, theme.PlaceHolderColor())
		sub.TextSize = theme.Size(theme.SizeNameCaptionText)
		objects = append(objects, head, sub)
	} else {
		objects = append(objects, head)
	}

	row := container.NewBorder(nil, nil, nil, control, container.NewPadded(container.NewVBox(objects...)))
	return settingsCard(row)
}
