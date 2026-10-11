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

> **Pre-alpha.** The UI and all six API methods are implemented and the HTTP
> layer is unit-tested, but the client is aligned with a backend that still
> lacks the channel-list and message-history routes. See
> [Backend requirements](#backend-requirements) before expecting a working
> chat round-trip.

## Features

- **Register / login** — email + password, JWT stored in the shared client.
- **Channel list** — scrollable list, create a channel with a type, name and
  avatar URL, refresh after creation.
- **Chat view** — message history, send with the button or `Enter`, live
  messages pushed over a WebSocket.
- **Logout** — clears the token and returns to the login view.
- **Single window, view swapping** — `Login ⇄ Register → Channels → Chat`.
- **Correct threading** — every HTTP call runs on a goroutine and every UI
  mutation goes through `fyne.Do`.
- **Configurable backend URL** — point the app at any host with
  `FYNCHAT_API_URL` instead of editing code.

## Screens

```
   ┌───────────┐        ┌──────────────┐        ┌─────────────┐
   │   Login   │ <────► │  Register    │        │  Channels   │
   │ email/pw  │        │ name/user/   │        │  ▸ general  │
   └─────┬─────┘        │ email/pw     │        │  ▸ random   │
         │              └──────┬───────┘        │  [+ New]    │
         └──────────────────────┘               │  [ Logout]  │
                     on success                 └──────┬──────┘
                                                    │ click
   ┌────────────────────────────────────────────────▼──────────┐
   │ ‹ Back            # general                                  │
   │─────────────────────────────────────────────────────────────│
   │ ada: hello there                                           │
   │ lin: hi!                                                   │
   │─────────────────────────────────────────────────────────────│
   │ [ Type a message...                    ] [ Send ]            │
   └────────────────────────────────────────────────────────────┘
```

## Project layout

One file per view, as the project convention dictates — no framework, no
state-management library, just structs, fields and callbacks.

| File | Contents |
| --- | --- |
| `main.go` | App + window bootstrap, shared `HTTPClient`, view wiring |
| `httpclient.go` | `HTTPClient`, `Channel`, `Message`, the six API methods |
| `login.go` | `loginView(...)` — email/password form |
| `register.go` | `registerView(...)` — name/username/email/password form |
| `channels.go` | `channelsView(...)` — channel list, new-channel dialog, logout |
| `chat.go` | `chatView(...)` — history, composer, WebSocket live messages |
| `httpclient_test.go` | `httptest`-based tests for every HTTP method |
| `docs/websockets.md` | Hub / client-pump / broadcast design doc (server-side) |

Navigation is done by passing callbacks. `main.go` owns the transitions:

```go
Login ⇄ Register → Channels → Chat
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
  starts but every request fails with a connection error dialog.

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

Details:

- Authenticated requests always carry `Authorization: Bearer <token>`, which
  is what `middlewares.Auth` expects.
- The token is read from `token` **or** `access_token`, so the client works
  with both the current handlers and the original API spec.
- Non-2xx responses become Go errors that include the status code and a
  512-byte snippet of the response body, and every error is surfaced with
  `dialog.ShowError` rather than being swallowed.

### Backend requirements

Two of the six routes do not exist in `fynchat-backend` yet. Until they are
added, channel history and message history will fail with a 404 dialog; login,
register and channel creation work as-is.

| Route the client needs | Backend status | Change required |
| --- | --- | --- |
| `GET /api/channels` | ❌ missing | add a list handler returning channel rows including `id` |
| `GET /api/channels/:id/messages` | ❌ missing | add a history handler returning message rows |
| `POST /api/message/create/:channel_id` | ⚠️ registered without the param | `MessageHandler.Create` reads `c.Param("channel_id")`, so the route must be `api.POST("/message/create/:channel_id", messagesHandler.Create)` — as registered today it always 400s on an empty channel id |
| `ws://…/ws` | ❌ missing | see [`docs/websockets.md`](docs/websockets.md) |
| Port | ⚠️ `SERVER_PORT` defaults to `8000` | run with `FYNCHAT_API_URL=http://localhost:8000` |

Also note, when you build the backend itself:

- its `go.mod` declares `module fynchat.com` while the code imports
  `fynchat/internal/...`, so it does not compile as committed;
- `routes.Setup` takes a `messagesHandler` argument that `cmd/fynchat/main.go`
  does not pass.

Both belong to the backend repo — this README only records them.

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
endpoint is absent the dialog is shown once and the view keeps working over
REST.

**Server side:** [`docs/websockets.md`](docs/websockets.md) is the full Hub /
client-pump / broadcast design to port into `fynchat-backend`, including
heartbeats, reconnect backoff and the Redis Pub/Sub path for multi-instance
deployments.

## Testing

```bash
go test -race ./...
```

`httpclient_test.go` runs the client against `net/http/httptest` servers and
covers: login, register, the backend's `token` field, error responses, the
`Bearer` header, request bodies and paths, list/create channel, list/send
message, and malformed JSON. No backend and no database needed — but the
system packages from [Requirements](#requirements) are, because `go test`
compiles the whole `main` package including the Fyne views.

Expected result:

```
ok  	fynchat	1.034s
```

`gofmt -l .` and `go vet ./...` are clean.

## Cross-compiling

Fyne cross-compiles from Linux through [fyne-cross](https://github.com/fyne-io/fyne-cross):

```bash
go install fyne.io/tools/cmd/fyne-cross@latest
fyne-cross windows   # -> fynchat.exe
fyne-cross darwin
```

Scratch output lands in `fyne-cross/tmp/` (git-ignored).

## Roadmap

- [ ] Backend: add `GET /api/channels` and `GET /api/channels/:id/messages`,
      and fix the `:channel_id` param on `POST /api/message/create` — see
      [Backend requirements](#backend-requirements).
- [ ] Implement the Hub described in `docs/websockets.md` in the backend.
- [ ] Reconnect with exponential backoff + re-fetch after reconnect.
- [ ] Refresh-token support (`/api/request` in the backend) so long sessions
      do not die on an expired JWT.
- [ ] Persist the token so the app reopens on the channel list.
- [ ] Message grouping, avatars, timestamps and `channel_type` handling in the
      channel list.
- [ ] `fynproto` (FCP) transport as the long-term replacement for REST + WS.

## Contributing

Issues and PRs are welcome. Please keep the style of the existing code: one
file per view, `fyne.Do` around every goroutine-driven UI update, no ignored
errors, and no dependencies outside core Fyne unless there is a strong reason.

## License

[MIT](LICENSE) © 2026 FynChat