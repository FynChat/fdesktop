package main

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/coder/websocket"
)

// chatView builds the chat screen for one channel. It shows the channel
// name on top, a scrollable message list in the middle, and a text entry +
// send button at the bottom. onBack returns to the channel list.
func chatView(client *HTTPClient, w fyne.Window, channelID, channelName string, onBack func()) fyne.CanvasObject {
	title := widget.NewLabel(channelName)

	messagesBox := container.NewVBox()
	scroll := container.NewVScroll(messagesBox)

	// render replaces the message labels with the given messages and
	// scrolls to the bottom.
	render := func(messages []Message) {
		messagesBox.RemoveAll()
		for _, m := range messages {
			messagesBox.Add(widget.NewLabel(m.Content))
		}
		messagesBox.Refresh()
		scroll.ScrollToBottom()
	}

	// load fetches messages for this channel from the backend.
	load := func() {
		go func() {
			messages, err := client.ListMessages(channelID)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				render(messages)
			})
		}()
	}
	load()

	// openWS connects to the backend WebSocket so new messages pushed by the
	// server appear live. The REST POST remains the write path; this is
	// receive-only. Closing the connection happens via cancel() on Back.
	wsCtx, cancelWS := context.WithCancel(context.Background())
	go func() {
		wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws?" + url.Values{
			"channel_id": {channelID},
			"token":      {client.Token},
		}.Encode()
		conn, _, err := websocket.Dial(wsCtx, wsURL, nil)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, w)
			})
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
				fyne.Do(func() {
					messagesBox.Add(widget.NewLabel(m.Content))
					messagesBox.Refresh()
					scroll.ScrollToBottom()
				})
			}
		}
	}()

	entry := widget.NewEntry()
	entry.SetPlaceHolder("Type a message...")

	// send delivers the typed message and refreshes the list.
	send := func() {
		content := entry.Text
		if content == "" {
			return
		}
		go func() {
			_, err := client.SendMessage(channelID, content)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				entry.SetText("")
				load()
			})
		}()
	}

	sendBtn := widget.NewButton("Send", send)
	entry.OnSubmitted = func(string) { send() }

	backBtn := widget.NewButton("Back", func() {
		cancelWS()
		onBack()
	})

	return container.NewBorder(
		container.NewVBox(backBtn, title),
		container.NewBorder(nil, nil, nil, sendBtn, entry),
		nil, nil,
		scroll,
	)
}
