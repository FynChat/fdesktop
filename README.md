# fdesktop – FynChat Qt Desktop

Qt 6 desktop client for FynChat using the custom FCP binary protocol.

## Overview

`fdesktop` provides a modern, Telegram-inspired chat UI with a collapsible icon tray, Catppuccin Mocha dark theme, and bubble-style message view. The client connects to a FynChat server via TCP and implements the shared FCP protocol defined in `include/shared/protocol.hpp`.

### Current architecture

* **Protocol layer** – `include/shared/protocol.hpp`
  `FcpHeader` + `CommandType` enum. Shared with fynproto. Serialisation is little-endian with explicit length framing.
* **Network client** – `src/fynclient.{hpp,cpp}`
  `QTcpSocket` wrapper with bounded allocation, partial-write loop and `MAX_PACKET` guard. Emits `connectedChanged(bool)` and `messageReceived(QByteArray)`.
* **UI shell** – `src/mainwindow.{hpp,cpp}`
  Dark themed `QMainWindow` with sidebar, chat history and input area.
* **Improved UI** – `src/mainwindow_improved.{hpp,cpp}` / `src/logindialog_improved.{hpp,cpp}`
  Collapsible icon tray, HTML bubble rendering, card-style login dialog. Implements the UX spec in `docs/Main.md`.
* **Auth dialog** – `src/logindialog.{hpp,cpp}`
  Modal login / register. Note: auth flow needs server acknowledgement before accepting – see Known Issues.
* **App entry** – `src/main.cpp`
  Creates `FynClient` for `127.0.0.1:8080` and launches the UI.

## Build

```bash
mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build .
```

Requirements:
* CMake ≥ 3.16
* Qt 6 Core, Network, Widgets
* C++17

Run:
```bash
./build/fdesktop
```

## UI Design

Design spec: `docs/Main.md`

* Color palette – Catppuccin Mocha
  * Background `#1e1e2e`, Surface `#11111b`
  * Accent `#89b4fa`, User bubble `#cba6f7`, Incoming `#313244`
  * Text primary `#cdd6f4`, secondary `#6c7086`
* Layout – left icon tray → collapsible chat sidebar → dynamic header → Telegram-style message area → input bar with emoji / attach / send
* Styling – Qt Style Sheets, 6-12px border radius, hover states

## Docs
```bash
FYNCHAT_API_URL=http://localhost:8000 go run .   # backend default port
```

## API contract

The client calls the routes `fynchat-backend` actually serves. Paths live in
named constants at the top of `httpclient.go`, so aligning with a future
backend change is a one-line edit:

| Client method | Request | Auth | Expected response |
| --- | --- | --- | --- |
| `Login` | `POST /login` `{email, password}` | – | `{"token": "...", "user": {...}}` |
| `Register` | `POST /register` `{name, username, email, password}` | – | `{"token": "...", "user": {...}}` |
| `ListChannels` | `GET /api/channels` | Bearer | `[{"id", "channel_name", "channel_avatar"}, ...]` |
| `CreateChannel` | `POST /api/channel/create` `{channel_type, name, avatar_url}` | Bearer | `{"channel_name", "channel_avatar"}` |
| `ListMessages` | `GET /api/channels/{id}/messages` | Bearer | `[{"id", "channel_id", "sender_id", "content", "created_at"}, ...]` |
| `SendMessage` | `POST /api/message/create/{id}` `{content}` | Bearer | the created message |
| `ListContacts` | `GET /api/contacts` | Bearer | `[{"id", "name", "username", "avatar_url"}, ...]` |
| `AddContact` | `POST /api/contacts` `{username}` | Bearer | the created contact, `201` |
| `OpenPrivateChat` | `POST /api/channel/create` `{channel_type:"p2p", peer_id}` | Bearer | the channel, existing or new |

Details:

- Authenticated requests always carry `Authorization: Bearer <token>`, which
  is what `middlewares.Auth` expects.
- The token is read from `token` **or** `access_token`, so the client works
  with both the current handlers and the original API spec.
- Non-2xx responses become Go errors that include the status code and a
  512-byte snippet of the response body, and every error is surfaced with
  `dialog.ShowError` rather than being swallowed.
- **Channel avatars travel as bytes.** The dialog reads the chosen image file
  into `[]byte`, and `CreateChannel` sends those bytes under `avatar_url`. On
  the way back in, `Channel.UnmarshalJSON` accepts `channel_avatar` as either a
  plain string or base64 (Go's default for `[]byte`), because the backend may
  return either. A channel with no image just gets `nil` and the UI falls back
  to an initials circle.

### Private chats are get-or-create

A private chat is identified by the pair of people in it, so there can only
ever be one. `OpenPrivateChat` therefore asks the backend for the chat with a
person and does not care whether that meant making one:

- no chat with that person yet (or it was deleted) → the backend creates it
- a chat already exists → the backend returns **the same channel**, and the
  client opens it

The client relies on this and does not check first: two requests must never
produce two chats with the same person. `POST /api/channel/create` with
`channel_type: "p2p"` must be idempotent per user pair.

A private chat is also the only kind the client does **not** name. The form
that creates a channel offers Group and Channel only — a private chat starts
from a contact, and the backend names it after the two people in it.

### Contacts and channels are separate

Adding a person and creating a conversation are different actions, and the UI
keeps them apart:

| Screen | Does | Route |
| --- | --- | --- |
| Contacts (`contacts.go`) | lists people, adds one by username | `GET`/`POST /api/contacts` |
| New channel (`dialog.go`) | names a group or a broadcast | `POST /api/channel/create` |

`POST /api/contacts` takes a **username**, not a display name: a handle is
what resolves to an account, and letting the client pick the name is how one
person becomes two contacts. The display name in the response is the
backend's. The client sends the handle with any leading `@` already stripped.

### Backend requirements

| Route the client needs | Backend status | Change required |
| --- | --- | --- |
| `GET /api/channels` | ✅ implemented | – |
| `GET /api/channels/:id/messages` | ✅ implemented | – |
| `POST /api/message/create/:channel_id` | ✅ implemented | – |
| `ws://…/ws` | ✅ implemented | `internal/realtime` — see [`docs/websockets.md`](docs/websockets.md) |
| `GET /api/contacts` | ✅ implemented | return the user's contacts |
| `POST /api/contacts` `{username}` | ❌ missing | resolve the handle, reject unknown users and duplicates |
| p2p `POST /api/channel/create` | ❌ missing | accept `peer_id`, make it get-or-create per pair |
| Port | ⚠️ `SERVER_PORT` defaults to `8000` | run with `FYNCHAT_API_URL=http://localhost:8000` |

The backend's module path also had to be corrected (`fynchat.com` vs
`fynchat/...`), and `routes.Setup` was being called without the
`messagesHandler` it requires. Both belong to the backend repo — this README
only records them.

The client and the backend were verified end to end against a live database:
login, channel list, message history, message send and live push all decode
correctly.

## WebSockets

`chat.go` opens **one connection per open chat** to:

```
ws://<host>/ws?channel_id=<id>&token=<jwt>
```

and appends every pushed `Message` JSON object to the list. REST stays the
write path — sending is a normal `POST`, the socket is receive-only. Messages
are de-duplicated by `Message.ID`, so the message you send is not rendered
twice when the server echoes it back.

The connection is closed when you press **Back** or close the window. If the
endpoint is absent the view keeps working over REST — live updates are a
bonus, not the load-bearing path.

**Server side:** `fynchat-backend/internal/realtime` implements the Hub /
client-pump / broadcast design from
[`docs/websockets.md`](docs/websockets.md). The socket authenticates from the
`token` query parameter rather than a header, because a WebSocket handshake
cannot carry one — so `/ws` sits outside the `/api` group on purpose.

## Testing

```bash
go test -race ./...
```

`httpclient_test.go` runs the client against `net/http/httptest` servers and
covers: login, register, the backend's `token` field, error responses, the
`Bearer` header, request bodies and paths, list/create channel, list/send
message, listing and adding contacts, get-or-create for private chats, and
malformed JSON. No backend and no database needed — but the system packages
from [Requirements](#requirements) are, because `go test` compiles the whole
`main` package including the Fyne views.

`dialog_test.go` covers the rules the UI can check without a server: what a
channel name may be, what counts as a username, and how contacts are searched.

Expected result:

```
ok  	fynchat	1.034s
```

`gofmt -l .` and `go vet ./...` are clean.

`make shots` renders every view to a PNG with the Fyne software driver, which
is how the UI gets reviewed without opening a window (`shot-shell-chat.png`,
`shot-contacts.png`, the dialogs, and so on). It sits behind a `screenshot`
build tag because Fyne's test driver runs `fyne.Do` inline rather than
queueing it, so the race detector fires on the harness even though the app is
correct. It also runs single-threaded and retries, for the same reason.

### Fyne 2.8 pitfalls this code works around

Each of these was found by a crash, not by guessing, and each has a comment at
the place it applies:

| Symptom | Cause | What the code does |
| --- | --- | --- |
| `slice bounds out of range` in `drawOblongStroke` | an `Entry` inside a container built hidden and shown later: it was never laid out, so its border is stroked at a negative radius | swaps a container's `Objects` (`shell.go`, `menupanel.go`) or uses its own dialog instead of hiding a form (`dialog.go`) |
| the same, from a list | a `widget.List` wrapped in a scroll: it has its own scroller, and an outer one hands it unbounded height | `SetMinSize` on a transparent rectangle inside `Max` (`contacts.go`) |
| a row of blank boxes | assigning `Text` without `Refresh()`, so the renderer keeps painting the old text | `Refresh()` after setting text |
| a search box stretched down the window | `NewBorder`'s left slot is for a side panel, not a top row | search entries go in the top region |
| nothing painted at all | the canvas walks children by asserting `*fyne.Container`, so a struct embedding one paints nothing | rows hold a plain container (`channelRow`, `contactRow`) |
| every tap on the sidebar is ignored | a `Stack` hands **every** child the full size, so an invisible `Button` meant to close the menu covered the whole window and swallowed every click | the scrim is a single tappable `scrimWidget` (`menupanel.go`), hidden until the menu opens |

## Cross-compiling

Fyne cross-compiles from Linux through [fyne-cross](https://github.com/fyne-io/fyne-cross):

```bash
go install fyne.io/tools/cmd/fyne-cross@latest
fyne-cross windows   # -> fynchat.exe
fyne-cross darwin
```

* `docs/FCP.md` – FCP protocol overview
* `docs/Main.md` – UI design & architecture specification
* `docs/Protocol_Integration.md` – fcproto-aligned specification and integration notes

## Known issues / TODO

* Authentication currently closes dialog on send without waiting for server ack. `authSucceeded` signal is declared but unused. Auth state machine needs to be wired.
* Credentials are sent in clear text over TCP. TLS and hashing required for production.
* Avatar upload has no size limit before base64 allocation.
* Packet parsing is duplicated between `FynClient` and `main.cpp`. `FynClient` should emit parsed `CommandType` + payload.
* `include/` should not be git-ignored. The shared protocol header must be versioned.
* Add reconnect logic, error surfacing, and bounded input.

## Contributing

* Keep serialization single-source-of-truth. Avoid `#pragma pack` reliance across compilers.
* Use Qt parent-child ownership; avoid raw owning pointers.
* Add `-Wall -Wextra -Wpedantic` to CMake and treat warnings as errors in CI.
* Run `test_ui_build.sh` before opening PRs.

## License

See repository policy.
[MIT](LICENSE) © 2026 FynChat
