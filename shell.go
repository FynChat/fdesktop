package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// shell builds the two-pane window the whole app lives in: the channel list on
// the left, the open chat on the right. Picking a chat swaps only the right
// pane, so the list stays put — the thing that makes Telegram Desktop feel
// quick is not going back to a list on every conversation.
//
// It returns the window content, the function that opens a chat in its
// right-hand pane (so the caller can reopen the last one on launch), and a stop
// function that unregisters the app-wide shortcuts and tears down the open
// chat. The stop must be called when the shell is replaced — on logout —
// or a Ctrl+N would still fire into the login form.
func shell(client *HTTPClient, prefs *Prefs, w fyne.Window, onLogout func()) (fyne.CanvasObject, func(Channel), func()) {
	// A Stack, not a Center: Center hugs its content, so the chat would sit
	// in the middle of the pane at its minimum size instead of filling it.
	content := container.NewStack(emptyChatState())

	// stopChat tears down the view being replaced. Without it, swapping
	// channels would leave the previous WebSocket goroutine running.
	var stopChat = func() {}

	openChat := func(ch Channel) {
		stopChat()

		var view fyne.CanvasObject
		view, stopChat = chatView(client, prefs, w, ch.ID, ch.ChannelName, ch.ChannelType, ch.ChannelAvatar)
		prefs.SetLastChannel(ch.ID)

		content.Objects[0] = view
		content.Refresh()
	}

	// The sidebar needs to highlight a row when its chat is opened, and it
	// only exists after openChat does, so it is filled in below. The same goes
	// for the menu, which the sidebar's hamburger button opens.
	var side sidebar
	var menu *sideMenu

	onSelect := func(ch Channel) {
		side.markOpen(ch)
		openChat(ch)
	}

	// The left pane shows one screen at a time — the chat list or the contacts.
	//
	// The two screens are SWAPPED rather than shown and hidden. Fyne 2.8
	// crashes the painter when an Entry lives inside a container that was built
	// hidden and revealed later (the entry is never laid out, so its border is
	// stroked at a negative radius), and both screens contain an Entry. Putting
	// in only what should be on screen avoids the whole problem — and it is
	// what the type picker in the new-channel dialog does too.
	pane := container.NewStack()

	showChats := func() {
		pane.Objects = []fyne.CanvasObject{side.view}
		pane.Refresh()
		if menu != nil {
			menu.showChannelSearch(true)
		}
	}
	showContacts := func() {
		pane.Objects = []fyne.CanvasObject{
			contactsView(client, w, func(ch Channel) {
				showChats()
				side.markOpen(ch)
				openChat(ch)
			}, showChats),
		}
		pane.Refresh()
		if menu != nil {
			menu.showChannelSearch(false)
		}
	}

	// Settings is a screen, not a dialog: it has a back button and sits in the
	// left pane the same way Contacts does, so the conversation beside it does
	// not change under the user.
	showSettings := func() {
		pane.Objects = []fyne.CanvasObject{
			settingsView(client, prefs, w, showChats, onLogout),
		}
		pane.Refresh()
		if menu != nil {
			menu.showChannelSearch(false)
		}
	}

	side = sidebarView(client, prefs, w, onSelect, onLogout, func() {
		menu.toggle(w.Canvas().Size())
	})
	showChats()

	menu = newSideMenu(client, prefs, side, onLogout, nil, showContacts, showSettings)

	// A draggable split, like the divider in Telegram Desktop: the list keeps
	// roughly a third of the window and both sides can be resized.
	split := container.NewHSplit(pane, content)
	split.Offset = 0.3

	// The menu floats above everything, parked off screen to the left until
	// the hamburger opens it.
	holder := container.NewWithoutLayout(menu.panel)
	holder.Resize(fyne.NewSize(menuWidth, 1))
	menu.panel.Move(fyne.NewPos(-menuWidth, 0))

	overlay := container.NewStack(split)
	overlay.Objects = append(overlay.Objects, menu.scrim, holder)

	// The app-wide shortcuts. Ctrl+Enter belongs to the composer and is bound
	// in chatView, because it only means anything while a chat is open; these
	// four mean the same thing wherever you are in the shell.
	//
	// Each is registered on the window canvas and unregistered by the returned
	// stop, so logging out does not leave a shortcut firing into a login form.
	var bound []fyne.Shortcut
	bind := func(def shortcutDef, handler func(fyne.Shortcut)) {
		sc := def.shortcut()
		w.Canvas().AddShortcut(sc, handler)
		bound = append(bound, sc)
	}

	bind(newChannelShortcut, func(fyne.Shortcut) {
		if side.newChannel != nil {
			side.newChannel()
		}
	})

	// Ctrl+F opens the menu and lands in the search box, which lives inside
	// the menu rather than the list header. Focus goes through the canvas:
	// an Entry has no Focus method of its own.
	bind(focusSearchShortcut, func(fyne.Shortcut) {
		if !menu.open {
			menu.toggle(w.Canvas().Size())
		}
		w.Canvas().Focus(side.search)
	})

	bind(openSettingsShortcut, func(fyne.Shortcut) { showSettings() })
	bind(helpShortcut, func(fyne.Shortcut) { showShortcutsDialog(w) })

	stopShell := func() {
		for _, sc := range bound {
			w.Canvas().RemoveShortcut(sc)
		}
		stopChat()
	}

	// Opening a chat from anywhere else, such as restoring the last one on
	// launch, still marks the row.
	return overlay, func(ch Channel) {
		side.markOpen(ch)
		openChat(ch)
	}, stopShell
}

// emptyChatState is what the right pane shows before a chat is picked.
func emptyChatState() fyne.CanvasObject {
	mark := appLogo(64)

	hint := widget.NewLabel("Pick a conversation on the left,\nor create a new channel to get going.")
	hint.Alignment = fyne.TextAlignCenter
	hint.Importance = widget.LowImportance

	return container.NewCenter(
		container.NewVBox(
			container.NewCenter(mark),
			container.NewCenter(hint),
		),
	)
}
