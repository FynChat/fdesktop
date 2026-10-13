package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// shortcutDef is one keyboard shortcut: the keys that trigger it and what they
// do. The list backs both the registration and the help dialog, so the two
// cannot drift apart — adding a shortcut here is what makes it work and shows
// it in the help.
type shortcutDef struct {
	name     string // shown in the help dialog, e.g. "Ctrl + Enter"
	desc     string
	key      fyne.KeyName
	modifier fyne.KeyModifier
}

// shortcut builds the Fyne object a view registers on its canvas.
func (s shortcutDef) shortcut() *desktop.CustomShortcut {
	return &desktop.CustomShortcut{KeyName: s.key, Modifier: s.modifier}
}

// The app's shortcuts. Enter alone starts a new line in the composer — Fyne's
// MultiLineEntry treats plain Enter as a newline and only fires OnSubmitted on
// Shift+Enter — so sending is bound to Ctrl+Enter by hand, the same keys
// Telegram Desktop uses.
var (
	sendMessageShortcut = shortcutDef{
		name:     "Ctrl + Enter",
		desc:     "Send the message you have typed",
		key:      fyne.KeyReturn,
		modifier: fyne.KeyModifierControl,
	}
	newChannelShortcut = shortcutDef{
		name:     "Ctrl + N",
		desc:     "Create a new channel",
		key:      fyne.KeyN,
		modifier: fyne.KeyModifierControl,
	}
	focusSearchShortcut = shortcutDef{
		name:     "Ctrl + F",
		desc:     "Search your channels",
		key:      fyne.KeyF,
		modifier: fyne.KeyModifierControl,
	}
	openSettingsShortcut = shortcutDef{
		name:     "Ctrl + ,",
		desc:     "Open settings",
		key:      fyne.KeyComma,
		modifier: fyne.KeyModifierControl,
	}
	helpShortcut = shortcutDef{
		name:     "Ctrl + /",
		desc:     "Show this list of shortcuts",
		key:      fyne.KeySlash,
		modifier: fyne.KeyModifierControl,
	}
)

// appShortcuts is every shortcut in one place, in the order the help dialog
// lists them.
func appShortcuts() []shortcutDef {
	return []shortcutDef{
		sendMessageShortcut,
		newChannelShortcut,
		focusSearchShortcut,
		openSettingsShortcut,
		helpShortcut,
	}
}

// showShortcutsDialog lists the keyboard shortcuts. It is how they are
// discovered: a shortcut nobody knows about may as well not exist.
func showShortcutsDialog(w fyne.Window) {
	rows := make([]fyne.CanvasObject, 0, len(appShortcuts()))
	for _, s := range appShortcuts() {
		keys := canvas.NewText(s.name, theme.ForegroundColor())
		keys.TextStyle = fyne.TextStyle{Bold: true}

		// The description goes in the Border's centre, not its right slot:
		// the right slot hugs each label at its own width and pins it to the
		// trailing edge, so the lines end up flush right and visually detached
		// from the keys they belong to.
		rows = append(rows, container.NewBorder(nil, nil, keys, nil, widget.NewLabel(s.desc)))
	}

	d := dialog.NewCustom("Keyboard shortcuts", "Close",
		container.NewPadded(container.NewVBox(rows...)), w)
	d.Resize(fyne.NewSize(420, 320))
	d.Show()
}
