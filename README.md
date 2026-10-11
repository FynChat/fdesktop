The Prompt

> See `docs/websockets.md` for the WebSocket (hub/client/broadcast) guide used
> to add live message push to the chat view.

    I'm building a desktop chat app with Fyne (fyne.io/fyne/v2) in Go. The backend is already running at http://localhost:8080. I need you to write the frontend code, one view at a time, in separate files.

    Backend API (already implemented):

        POST /register — body: {"name": "...", "username": "...", "email": "...", "password": "..."} → returns {"access_token": "...", "user": {...}}

        POST /login — body: {"email": "...", "password": "..."} → returns {"access_token": "...", "user": {...}}

        GET /channels — header: Authorization: Bearer <token> → returns [{"id": "uuid", "channel_name": "...", "channel_avatar": "..."}]

        POST /channels — header: Authorization: Bearer <token> — body: {"channel_type": "...", "name": "...", "avatar_url": "..."} → returns {"channel_name": "...", "channel_avatar": "..."}

        GET /channels/{id}/messages — header: Authorization: Bearer <token> → returns [{"id": "uuid", "channel_id": "uuid", "sender_id": 1, "content": "...", "created_at": "..."}]

        POST /channels/{id}/messages — header: Authorization: Bearer <token> — body: {"content": "..."} → returns the created message

    What I want you to write:

    1. A httpClient package or file — an HTTPClient struct with a Token field and methods:

        Login(email, password string) (string, error) — returns the token

        Register(name, username, email, password string) (string, error) — returns the token

        ListChannels() ([]Channel, error)

        CreateChannel(channelType, name, avatar string) (Channel, error)

        ListMessages(channelID string) ([]Message, error)

        SendMessage(channelID, content string) (Message, error)

    Use http.NewRequest + http.Client.Do. Always set the Authorization: Bearer <token> header on authenticated requests. Handle the error from every call. Parse the JSON response into structs.

    2. login.go — a Fyne view with:

        Two entries: email, password (password hidden)

        One button: Login

        A link or button to switch to Register

        On click: call HTTPClient.Login. On success, store the token and call a onLoginSuccess() callback. On failure, show dialog.ShowError.

    3. register.go — same, but with name, username, email, password fields.

    4. channels.go — a Fyne view with:

        A scrollable list of channels (widget.NewList or container.NewVScroll + widget.NewLabel per channel)

        A "New Channel" button that opens a dialog.ShowForm for channel_type, name, avatar_url

        On load: call ListChannels. On error, show a dialog.

        On channel click: call an onChannelSelected(channelID) callback.

        A "Logout" button that clears the token and returns to login.

    5. chat.go — a Fyne view with:

        Top: a label showing the channel name

        Middle: a scrollable list of messages (container.NewVScroll + a widget.NewVBox container with message labels)

        Bottom: a widget.NewEntry for typing + a widget.NewButton for sending

        On load: call ListMessages. On submit: call SendMessage, then refresh the list.

        A "Back" button that returns to the channel list.

    6. main.go — a Fyne app that:

        Creates a fyne.App and one window

        Holds the HTTPClient and current state (token, selected channel ID)

        Switches between views: Login → Channels → Chat

        Uses window.SetContent() to swap views

    Rules:

        Every UI update that happens from a goroutine (after an HTTP call) must be wrapped in fyne.Do or fyne.DoAndWait.

        Handle every error. No ignored errors.

        Use fyne.io/fyne/v2 and fyne.io/fyne/v2/widget, fyne.io/fyne/v2/container, fyne.io/fyne/v2/dialog.

        Don't use any Fyne community extensions unless necessary. Stick to core Fyne.

        One file per view.

        Keep the code simple. No over-engineering. No state management library. Just structs, fields, and callbacks.

    What I want from you first: Start with only the httpClient file. Write it completely. Then wait for my confirmation before writing the next file
