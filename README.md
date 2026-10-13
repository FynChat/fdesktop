<div align="center">
  <img src="Icon.png" alt="FynChat logo" width="96" height="96">

  # FynChat Desktop

  A cross-platform desktop chat client for the FynChat platform, built with Go and [Fyne](https://fyne.io).

  `frontend` · Go 1.26 · [Fyne](https://fyne.io) v2.8 · MIT License
</div>

---

## What this is

**fdesktop** is the desktop client of the FynChat messenger. It is a single
native window (not an Electron wrapper, not a browser) that talks to the
FynChat backend over HTTP and WebSockets.

This repository contains **only the GUI**. Accounts, channels, messages and
persistence live in two sibling repositories.

| Repository | Role |
| --- | --- |
| **[FynChat/fynchat-backend](https://github.com/FynChat/fynchat-backend)** | Go + Gin REST API, JWT auth, PostgreSQL persistence |
| **[FynChat/fynproto](https://github.com/FynChat/fynproto)** | FCP — binary real-time messaging protocol (codec, TCP transport, client/server examples) |
| **FynChat/fdesktop** *(this repo)* | Fyne desktop UI, the HTTP client and the WebSocket view |

The desktop app talks plain REST + WebSocket to `fynchat-backend`.
`fynproto` is the planned replacement for the HTTP/WebSocket transport once
FCP lands — see [Roadmap](#roadmap).

## Status

> **Pre-alpha.** The UI and every API method the client calls are implemented
> and unit-tested, and the client talks to `fynchat-backend` successfully:
> login, the channel list, message history and sending all work against a live
> database. Channel avatars are picked from disk and rendered in the list. Live
> message push needs a WebSocket the backend does not expose yet — see
> [Backend requirements](#backend-requirements).

## Features

- **Register / login** — checked in the form before anything is sent, JWT stored
  in the shared client. Both screens sit on a raised card so the form reads as
  one object instead of floating on the background.
- **Channel list** — avatars, channel type, create a channel, pull to refresh.
- **Chat view** — history as bubbles with sender and time, own messages on the
  right, consecutive messages grouped, date separators, live push over a
  WebSocket when the backend exposes one.
- **Channel avatars** — the new-channel dialog picks an image from your computer
  with a file selector (no URL to paste) and shows a live preview of how the
  channel will look in the list. Channels without an image fall back to a
  coloured circle with the first letter, the same trick Telegram uses.
- **Auto-growing message box** — the composer grows as you type, up to six rows,
  and shrinks back when the text is cleared.
- **Emoji and attachments** — an emoji palette inserts a reaction at the caret,
  and an attach button opens the system file picker (the backend has no upload
  route yet, so a picked file is acknowledged rather than silently dropped).
- **Dark and light** — the palette icon in the header switches, and the choice
  is remembered.
- **Stays where you left off** — the session, the last channel and the theme
  are stored in the app preferences, so the app reopens on the channel you had
  open instead of the login form.
- **Session expiry** — a rejected token drops you back to the login screen
  once, instead of failing every action with a dialog.
- **One window, two panes** — the chat list stays put while the conversation
  beside it changes, and the divider is draggable.
- **Slide-out menu that closes on tap** — the hamburger opens a menu over a dim
  scrim; tapping anywhere outside the menu closes it.
- **Correct threading** — every HTTP call runs on a goroutine and every UI
  mutation goes through `fyne.Do`.
- **Configurable backend URL** — point the app at any host with
  `FYNCHAT_API_URL` instead of editing code.

### Details worth knowing

- **Errors use a banner, not a dialog.** A failed send leaves your text in the
  box so you can retry instead of retyping.
- **Buttons lock while a request is in flight**, so a double click cannot send
  the same message twice.
- **The scroll only follows new messages when you are already at the bottom.**
  Reading older messages is not interrupted by someone talking.

## Screens

The app is one window split in two, the way Telegram Desktop is: the channel
list on the left, the open conversation on the right. Picking a chat swaps only
the pane beside the list, so you never navigate back to a list to change
conversation. The divider is draggable.

```
┌──────────────────────┬────────────────────────────────────────┐
│ FynChat           + ☰│  general                             🎨⟳│
├──────────────────────┤  Group                                    │
│ (G) general          │  ─────────────────────────────────────  │
│ (D) design-team    ▶ │              Yesterday                   │
│ (A) ada              │  (2) User #2            12:00            │
│                      │      morning everyone — finally green   │
│  ← open chat is      │  (3) User #3            09:00            │
│    filled with the   │      does the channel list still 404?   │
│    accent colour     │                  no — added yesterday   │
│                      │  ─────────────────────────────────────  │
│                      │  [ Message…                    📎 😊 ➤  │
└──────────────────────┴────────────────────────────────────────┘
```

Tapping **☰** slides out the menu over a dim scrim: the account row, the
channel search box, Contacts, and Settings. Tapping anywhere outside the menu
closes it.

Before signing in there is no list to show, so the auth screens take the whole
window — a centred card with the logo, the fields and one primary action.

## Project layout

One file per view, as the project convention dictates — no framework, no
state-management library, just structs, fields and callbacks.

| File | Contents |
| --- | --- |
| `main.go` | App + window bootstrap, shared `HTTPClient`, view wiring |
| `httpclient.go` | `HTTPClient`, `Channel`, `Message`, every API method, avatar bytes |
| `login.go` | `loginView(...)` + the shared auth card both screens use |
| `register.go` | `registerView(...)` — name/username/email/password form |
| `channels.go` | `sidebarView(...)` — the chat list column: search, rows, new channel |
| `shell.go` | the two-pane window, the menu overlay and the pane swap |
| `chat.go` | `chatView(...)` — header, history as bubbles, auto-growing composer, emoji picker |
| `dialog.go` | the new-channel modal: type picker, avatar file selector + live preview, inline errors |
| `menupanel.go` | the slide-out side menu: account row, search, settings, tap-to-close scrim |
| `contacts.go` | `contactsView(...)` — list people, add one by username |
| `ui.go` | shared widgets: avatars (initials or image), link labels, banners, icons |
| `theme.go` | the Telegram-style palette and the custom Fyne theme |
| `lottie.go` | lightweight looping animations (spinner, typing dots) via Fyne primitives |
| `demo.go` | `FYNCHAT_DEMO=1` sample data, served from memory |
| `httpclient_test.go` | `httptest`-based tests for every HTTP method |
| `session_test.go` | the session survives a restart and logout clears it |
| `validate_test.go` | form validation rules |
| `dialog_test.go` | channel-name and username rules |
| `ui_test.go` | the async button guard and the side-menu scrim |
| `shot_test.go` | renders each view to a PNG for review (`make shots`) |
| `docs/websockets.md` | Hub / client-pump / broadcast design doc (server-side) |

Navigation is done by passing callbacks. `main.go` owns the transitions:

```go
Login ⇄ Register → the two-pane window ⇄ Chat
Chat --Back--> Channels --Logout--> Login
```

## Requirements

- **Go 1.26+** (see `go.mod`).
- **A C toolchain** — Fyne is a cgo/OpenGL toolkit, so the GUI needs
  development headers at build time.

<details>
<summary>Linux system packages</summary>

Fyne needs OpenGL and windowing headers. Without them `go build` fails with
`pkg-config: executable file not found` or
`fatal error: wayland-client-core.h: No such file or directory`.

**Debian / Ubuntu**

```bash
sudo apt install build-essential pkg-config libgl1-mesa-dev libwayland-dev libxkbcommon-dev xorg-dev
```

**Fedora**

```bash
sudo dnf install gcc pkgconf-pkg-config mesa-libGL-devel wayland-devel libxkbcommon-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel
```

**Arch**

```bash
sudo pacman -S base-devel pkgconf mesa wayland libxkbcommon libx11 libxcursor libxrandr libxinerama libxi
```

</details>

- **A running `fynchat-backend`** (see the repo link above). Without it the app
  starts but every request fails with a connection error in the error banner.

## Demo mode

To look at the UI without a backend, a database or an account:

```bash
FYNCHAT_DEMO=1 go run .
```

It starts an in-process server that answers the same routes the real backend
does, with sample channels and a conversation long enough to show grouping,
date separators, both bubble styles and a wrapped message. The app takes the
same code path as with a real backend — nothing is mocked inside the views.

## Getting started

```bash
git clone git@github.com:FynChat/fdesktop.git
cd fdesktop
go mod download
go run .
```

Build a binary instead:

```bash
go build -o fynchat .
./fynchat
```

The app needs a display (X11 or Wayland). On a headless machine you will get a
GLFW error on startup — run it under `xvfb-run` if you only need a smoke test.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `FYNCHAT_API_URL` | `http://localhost:8080` | Base URL of the backend. Also used to derive the WebSocket URL (`http` → `ws`). |

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
| `GET /api/contacts` | ❌ missing | return the user's contacts |
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

The connection is torn down when you switch to another channel or close the
window — the view returns a stop function the shell calls before swapping. If
the endpoint is absent the view keeps working over REST — live updates are a
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
channel name may be, what counts as a username, how contacts are searched, and
that the demo server answers the client's routes. `validate_test.go` covers the
email and login/registration rules. `ui_test.go` covers the async button guard
and the side-menu scrim (that it starts hidden and closes on a tap).

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

Scratch output lands in `fyne-cross/tmp/` (git-ignored).

## Roadmap

- [ ] Backend: implement the Hub described in `docs/websockets.md`.
- [ ] Reconnect with exponential backoff + re-fetch after reconnect.
- [ ] Refresh-token support (`/api/request` in the backend) so long sessions
      do not die on an expired JWT.
- [x] Persist the token so the app reopens on the channel list.
- [x] Message grouping, avatars, timestamps and `channel_type` handling in the
      channel list.
- [ ] Backend: attachment upload endpoint, so the attach button can send files.
- [ ] Unread message tracking and badges in the channel list.
- [ ] `fynproto` (FCP) transport as the long-term replacement for REST + WS.

## Contributing

Issues and PRs are welcome. Please keep the style of the existing code: one
file per view, `fyne.Do` around every goroutine-driven UI update, no ignored
errors, and no dependencies outside core Fyne unless there is a strong reason.

## License

[MIT](LICENSE) © 2026 FynChat