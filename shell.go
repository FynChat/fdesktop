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
// It returns the window content and the function that opens a chat, so the
// caller can reopen the last one on launch.
func shell(client *HTTPClient, prefs *Prefs, w fyne.Window, onLogout func()) (fyne.CanvasObject, func(Channel)) {
	// A Stack, not a Center: Center hugs its content, so the chat would sit
	// in the middle of the pane at its minimum size instead of filling it.
	content := container.NewStack(emptyChatState())

	// stopChat tears down the view being replaced. Without it, swapping
	// channels would leave the previous WebSocket goroutine running.
	var stopChat = func() {}

	openChat := func(ch Channel) {
		stopChat()

		var view fyne.CanvasObject
		view, stopChat = chatView(client, prefs, w, ch.ID, ch.ChannelName, ch.ChannelType)
		prefs.SetLastChannel(ch.ID)

		content.Objects[0] = view
		content.Refresh()
	}

	// The sidebar needs to highlight a row when its chat is opened, and it
	// only exists after openChat does, so it is filled in below.
	var markOpen func(Channel)
	onSelect := func(ch Channel) {
		markOpen(ch)
		openChat(ch)
	}

	sidebar, markOpen := sidebarView(client, prefs, w, onSelect, onLogout)

	// A draggable split, like the divider in Telegram Desktop: the list keeps
	// roughly a third of the window and both sides can be resized.
	split := container.NewHSplit(sidebar, content)
	split.Offset = 0.3

	// Opening a chat from anywhere else, such as restoring the last one on
	// launch, still marks the row.
	return split, func(ch Channel) {
		markOpen(ch)
		openChat(ch)
	}
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
