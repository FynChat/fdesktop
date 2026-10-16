# fdesktop — FynChat Desktop

![fdesktop logo](resources/logo.png)

A Qt 6 desktop client for FynChat, built around the custom FCP binary protocol. The UI follows modern messenger conventions: a two-pane layout with a collapsible icon tray, bubble-style chat history, and a Catppuccin Mocha dark theme.

[![Version](https://img.shields.io/badge/version-0.2.0-89b4fa)](#build)
[![Platform](https://img.shields.io/badge/platform-Linux%20%C2%B7%20Debian%20%7C%20Ubuntu%20%C2%B7%20amd64-11111b)](#build)
[![Qt](https://img.shields.io/badge/Qt-6.6-green.svg)](https://doc.qt.io/qt-6/)
[![C++](https://img.shields.io/badge/C%2B%2B-17-blue.svg)](https://isocpp.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Overview

fdesktop connects to a FynChat server over TCP and speaks the shared FCP protocol
defined in `include/shared/protocol.hpp`. It provides:

* Two-pane layout with an icon tray and a collapsible chat sidebar
* Bubble message view with separate colours for incoming and outgoing messages
* Card-style login / register dialog that waits for the server's answer
* Catppuccin Mocha theme applied through Qt Style Sheets
* Strict packet parsing: bounded buffers, size limits, no unbounded allocations

Current components:

* `src/mainwindow.{hpp,cpp}` — main window: icon tray, sidebar, chat header, message history, composer
* `src/logindialog.{hpp,cpp}` — login / register dialog
* `src/fynclient.{hpp,cpp}` — `QTcpSocket` wrapper, framing and packet dispatch
* `src/main.cpp` — entry point, wires the client signals to the UI
* `include/shared/protocol.hpp` — `FcpHeader` and the `CommandType` catalogue

## Screenshots

*UI preview coming soon. The design spec is in `docs/Main.md`.*

## Build

fdesktop is built and packaged on Debian-based Linux distributions, 64-bit.

### Requirements

* Debian 12 / Ubuntu 22.04 or newer (amd64)
* CMake ≥ 3.16 and a C++17 compiler
* Qt 6 (Core, Network, Widgets)

### Install the dependencies

```bash
sudo apt update
sudo apt install build-essential cmake qt6-base-dev
```

### Compile

```bash
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j"$(nproc)"
```

### Run

```bash
./build/fdesktop
```

The client connects to `127.0.0.1:8080` on startup and opens the login dialog.

### Debian package

Release builds are packaged as `fynchat-desktop_<version>_amd64.deb`. The package
depends on the Qt 6 runtime libraries (`libqt6core6`, `libqt6widgets6`,
`libqt6network6`), which `apt` resolves automatically:

```bash
sudo apt install ./fynchat-desktop_0.2.0_amd64.deb
```

## UI Design

Design spec: `docs/Main.md`

Palette — Catppuccin Mocha:

* Background `#1e1e2e`, surface `#11111b`
* Accent `#89b4fa`, outgoing bubble `#cba6f7`, incoming bubble `#313244`
* Text primary `#cdd6f4`, secondary `#6c7086`

Layout: icon tray → collapsible chat sidebar → dynamic header → message area →
input bar with emoji, attach and send controls.

## Documentation

* `docs/FCP.md` — FCP protocol overview
* `docs/Main.md` — UI design and architecture spec
* `docs/Protocol_Integration.md` — protocol alignment and integration notes
* `changelog.txt` — release history

## Third-party

* Qt 6 — LGPL
* OpenSSL — Apache 2.0, used through Qt Network
* Catppuccin colour palette — MIT

## Known limitations

* The server address is fixed at `127.0.0.1:8080`; there is no settings screen yet
* Connections are plain TCP — credentials are not protected by TLS
* No automatic reconnection: restart the client if the server goes away
* The emoji button is a placeholder and does nothing yet
* Avatar uploads are capped at 5 MB

## Contributing

* Keep serialization in one place; avoid relying on `#pragma pack` across compilers
* Use Qt parent-child ownership instead of raw owning pointers
* Build with `-Wall -Wextra -Wpedantic` and keep the build warning-free
* Describe the change and the version impact in `changelog.txt`

## License

Released under the [MIT License](LICENSE).
