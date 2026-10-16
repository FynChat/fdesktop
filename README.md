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
