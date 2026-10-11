package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/coder/websocket"
)

// chatView builds the chat screen for one channel. It shows the channel name
// in a header, the message history as bubbles, and a composer pinned to the
// bottom. onBack returns to the channel list.
func chatView(client *HTTPClient, prefs *Prefs, w fyne.Window, channelID, channelName string, onBack func()) fyne.CanvasObject {
	messagesBox := container.NewVBox()
	banner := errorBannerWidget()

	// Remember this channel so the app can reopen it next launch.
	if prefs != nil {
		prefs.SetLastChannel(channelID)
	}

	// A short conversation should sit at the bottom of the window, not float
	// at the top like a list of files: the spacer above expands first and
	// pushes the messages down, exactly as Telegram does.
	anchored := container.NewVBox(layout.NewSpacer(), messagesBox)
	scroll := container.NewVScroll(anchored)

	// nearBottom reports whether the newest messages are already in view.
	// Auto-scrolling on every arrival yanks you down while you are reading
	// history, so the view only follows when you are already at the end.
	nearBottom := func() bool {
		offset := scroll.Offset.Y
		height := scroll.Size().Height
		content := scroll.Content.MinSize().Height
		return content-(offset+height) < scrollThreshold
	}

	// Refresh the parent, not messagesBox: messages are laid out by the
	// anchored VBox, so refreshing the inner box alone never resizes it.
	refresh := func(follow bool) {
		anchored.Refresh()
		if follow {
			scroll.ScrollToBottom()
		}
	}

	// rendered tracks the message IDs already on screen. A message arrives
	// twice when the user sends it: once through the REST response/refresh
	// and once through the WebSocket push from the server.
	rendered := make(map[string]bool)

	// appendMessage adds one message, skipping IDs that are already shown.
	appendMessage := func(m Message) {
		key := messageKey(m)
		if rendered[key] {
			return
		}
		rendered[key] = true
		messagesBox.Add(messageRow(m, client.UserID, false))
		refresh(nearBottom())
	}

	// render replaces the messages with the given history and scrolls to the
	// bottom, so the newest message is the one you see.
	render := func(messages []Message) {
		messagesBox.RemoveAll()

		// Record every id up front: the rows are grouped and split by
		// buildMessageRows, and a later live push of the same message must
		// still be recognised as one already on screen.
		rendered = make(map[string]bool)
		for _, m := range messages {
			rendered[messageKey(m)] = true
		}
		for _, row := range buildMessageRows(messages, client.UserID) {
			messagesBox.Add(row)
		}
		refresh(true)
	}

	// load fetches messages for this channel from the backend.
	load := func() {
		go func() {
			messages, err := client.ListMessages(channelID)

			fyne.Do(func() {
				if err != nil {
					banner.Show(err)
					return
				}
				banner.Clear()
				render(messages)
			})
		}()
	}
	load()

	// openWS connects to the backend WebSocket so new messages pushed by the
	// server appear live. The REST POST remains the write path; this is
	// receive-only. Closing the connection happens via cancel() on Back or
	// when the window is closed.
	wsCtx, cancelWS := context.WithCancel(context.Background())
	w.SetOnClosed(cancelWS) // don't leak the connection if the app is closed
	go func() {
		wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws?" + url.Values{
			"channel_id": {channelID},
			"token":      {client.Token},
		}.Encode()
		conn, _, err := websocket.Dial(wsCtx, wsURL, nil)
		if err != nil {
			// Live updates are a bonus: the view works over REST, so a missing
			// /ws endpoint must not interrupt the user with a dialog.
			return
		}
		defer conn.CloseNow()
		for {
			_, data, err := conn.Read(wsCtx)
			if err != nil {
				return // disconnected or view closed
			}
			var m Message
			if json.Unmarshal(data, &m) == nil {
				fyne.Do(func() { appendMessage(m) })
			}
		}
	}()

	// A multi-line composer, like Telegram's: Enter starts a new line and
	// Ctrl+Enter sends. A single-line entry cannot hold a paragraph and it is
	// the main thing people notice when chatting.
	entry := widget.NewMultiLineEntry()
	entry.SetPlaceHolder("Message")
	entry.Wrapping = fyne.TextWrapWord
	entry.SetMinRowsVisible(1)

	sending := widget.NewButtonWithIcon("", theme.MailSendIcon(), nil)
	sending.Importance = widget.HighImportance

	// send delivers the typed message and reloads the history.
	send := func() {
		content := strings.TrimSpace(entry.Text)
		if content == "" || sending.Disabled() {
			return
		}
		sending.Disable()

		go func() {
			_, err := client.SendMessage(channelID, content)

			fyne.Do(func() {
				sending.Enable()
				if err != nil {
					// Keep the text: the user can retry instead of
					// retyping what they just wrote.
					banner.Show(err)
					return
				}
				entry.SetText("")
				banner.Clear()
				load()
			})
		}()
	}

	sending.OnTapped = send
	entry.OnSubmitted = func(string) { send() }

	// Telegram uses a bare arrow rather than a labelled button.
	backBtn := iconButton(theme.NavigateBackIcon(), func() {
		cancelWS()
		onBack()
	})

	refreshBtn := iconButton(theme.ViewRefreshIcon(), func() {
		banner.Clear()
		load()
	})

	themeBtn := iconButton(theme.ColorPaletteIcon(), func() {
		if prefs == nil {
			return
		}
		prefs.SetDarkTheme(!prefs.DarkTheme())
		prefs.apply()
	})

	header := headerBar(channelName, themeBtn, refreshBtn, backBtn)
	composer := container.NewBorder(nil, banner.wrap, nil, sending, container.NewPadded(entry))

	return container.NewBorder(header, composer, nil, nil, container.NewPadded(scroll))
}

// scrollThreshold is how close to the end counts as "at the bottom", in
// pixels. A little slack means a slightly shorter window still counts.
const scrollThreshold = 48

// messageKey identifies a message for the de-duplication map, falling back to
// content when the backend sends no id.
func messageKey(m Message) string {
	if m.ID != "" {
		return m.ID
	}
	return m.Content + "|" + m.CreatedAt
}

// buildMessageRows turns a history into display rows. Telegram inserts a date
// separator when the day changes and groups consecutive messages from the
// same sender so the avatar and name are not repeated on every line.
func buildMessageRows(messages []Message, myUserID int) []fyne.CanvasObject {
	var rows []fyne.CanvasObject

	lastDay := ""
	for i, m := range messages {
		day := messageDay(m)
		if day != lastDay {
			rows = append(rows, daySeparator(day))
			rows = append(rows, messageRow(m, myUserID, false))
			lastDay = day
			continue
		}

		grouped := i > 0 && messages[i-1].SenderID == m.SenderID
		rows = append(rows, messageRow(m, myUserID, grouped))
	}
	return rows
}

// daySeparator is the centred "Today" / "11 Oct 2026" pill between days.
func daySeparator(day string) fyne.CanvasObject {
	text := canvas.NewText(day, theme.PlaceHolderColor())
	text.TextSize = theme.Size(theme.SizeNameCaptionText)

	pill := canvas.NewRectangle(pillBackground())
	pill.CornerRadius = 8
	pill.SetMinSize(fyne.NewSize(90, 22))

	return container.NewCenter(
		container.NewStack(pill, container.NewCenter(container.NewPadded(text))),
	)
}

// messageDay returns the calendar day of a message, or "" when the timestamp
// does not parse.
func messageDay(m Message) string {
	t, err := time.Parse(time.RFC3339, m.CreatedAt)
	if err != nil {
		return ""
	}
	switch t.Format("2006-01-02") {
	case time.Now().Format("2006-01-02"):
		return "Today"
	case time.Now().AddDate(0, 0, -1).Format("2006-01-02"):
		return "Yesterday"
	default:
		return t.Format("2 Jan 2006")
	}
}

// messageRow draws one chat bubble. Telegram keeps your own messages on the
// right in a muted accent colour and everyone else's on the left, with the
// sender's name inside the bubble. A grouped message keeps the bubble but
// drops the avatar and the repeated name.
func messageRow(m Message, myUserID int, grouped bool) fyne.CanvasObject {
	mine := myUserID != 0 && m.SenderID == myUserID

	// The API returns sender_id only, so other users are labelled by id
	// until the backend includes a name on the message.
	name, letter := "You", ""
	if !mine {
		name = fmt.Sprintf("User #%d", m.SenderID)
		letter = strconv.Itoa(m.SenderID)
	}

	// Name and timestamp live inside the bubble, above the text.
	meta := container.NewHBox()
	if !mine && !grouped {
		meta.Add(canvas.NewText(name, theme.ForegroundColor()))
		meta.Add(layout.NewSpacer())
	}
	if stamp := messageTime(m); stamp != "" {
		meta.Add(canvas.NewText(stamp, theme.PlaceHolderColor()))
	}
	for _, obj := range meta.Objects {
		if t, ok := obj.(*canvas.Text); ok {
			t.TextSize = theme.Size(theme.SizeNameCaptionText)
		}
	}

	// A Label (not canvas.Text) so long messages wrap instead of running off
	// the edge of the window. A wrapped label reports a tiny minimum width,
	// so the bubble rectangle carries the width instead: the Stack then hands
	// that width to the label and the text wraps inside it.
	body := widget.NewLabel(m.Content)
	body.Wrapping = fyne.TextWrapWord

	width := float32(150)
	if utf8.RuneCountInString(m.Content) > 24 {
		width = 340
	}

	bubble := canvas.NewRectangle(bubbleColor(mine))
	bubble.CornerRadius = 12
	bubble.SetMinSize(fyne.NewSize(width, 28))

	content := container.NewVBox(meta, body)
	stack := container.NewStack(bubble, container.NewPadded(content))

	if mine {
		return container.NewHBox(layout.NewSpacer(), stack)
	}
	if grouped {
		// Keep the avatar column empty so grouped bubbles stay aligned.
		gutter := container.NewGridWrap(fyne.NewSquareSize(36), canvas.NewRectangle(color.Transparent))
		return container.NewHBox(gutter, stack)
	}

	avatarBox := container.NewGridWrap(fyne.NewSquareSize(36), avatar(letter, 36))
	return container.NewHBox(avatarBox, stack)
}

// bubbleColor picks the fill for a message bubble: accent for your own
// messages, a quiet panel colour for everyone else's.
func bubbleColor(mine bool) color.Color {
	if mine {
		return outgoingBubble()
	}
	return incomingBubble()
}

// messageTime formats the RFC3339 timestamp the backend sends, returning ""
// when it does not parse so a bad value never breaks the row.
func messageTime(m Message) string {
	t, err := time.Parse(time.RFC3339, m.CreatedAt)
	if err != nil {
		return ""
	}
	return t.Format("15:04")
}
