# fdesktop — FynChat Desktop

![fdesktop logo](resources/logo.png)

A Qt 6 desktop client for FynChat built around the custom FCP binary protocol. The UI is inspired by modern messenger apps — clean two-pane layout, collapsible navigation tray, bubble chat with Catppuccin Mocha theming, and a responsive composer.

> Development is currently on branch `feat/ui-overhaul-avatars-auth`. Read the docs from that branch for the latest UI spec.

[![License](https://img.shields.io/badge/license-GPL--3.0-blue.svg)](LICENSE)
[![Qt](https://img.shields.io/badge/Qt-6-green.svg)](https://qt.io)
[![C++](https://img.shields.io/badge/C%2B%2B-17-blue.svg)](https://isocpp.org)

## Overview

fdesktop connects to a FynChat server via TCP and implements the shared FCP protocol defined in `include/shared/protocol.hpp`. The client provides:

* Two-pane Telegram-style layout with a collapsible icon tray and chat sidebar
* Bubble message view with user / incoming color coding
* Card-style login / register dialog with validation and busy states
* Catppuccin Mocha dark theme with QSS styling, rounded corners, and hover states
* Protocol layer shared with fynproto for consistent serialization

Current components:

* `include/shared/protocol.hpp` — `FcpHeader` + `CommandType` enum, little-endian framing
* `src/fynclient.{hpp,cpp}` — `QTcpSocket` wrapper with bounded allocation and `MAX_PACKET` guard
* `src/mainwindow_improved.{hpp,cpp}` / `src/logindialog_improved.{hpp,cpp}` — improved UI per `docs/Main.md`
* `src/main.cpp` — app entry, defaults to `127.0.0.1:8080`

## Screenshots

*UI preview coming soon. The design spec is in `docs/Main.md`.*

## Supported systems

Designed for Linux desktops. Builds tested on:

* GNU/Linux x86_64 with Qt 6
* Docker build for CentOS via `docker-compose.centos.yml`

Binary packages may be added later for Windows and macOS.

## Build

Requirements:

* CMake ≥ 3.16
* Qt 6 Core, Network, Widgets
* C++17 compiler

```bash
mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build .
```

Run:

```bash
./fdesktop
```

Docker build:

```bash
docker compose -f docker-compose.centos.yml build
```

## UI Design

Design spec: `docs/Main.md`

Palette — Catppuccin Mocha:

* Background `#1e1e2e`, Surface `#11111b`
* Accent `#89b4fa`, User bubble `#cba6f7`, Incoming `#313244`
* Text primary `#cdd6f4`, secondary `#6c7086`

Layout: left icon tray → collapsible chat sidebar → dynamic header → message area → input bar with emoji / attach / send.

## Documentation

* `docs/FCP.md` — FCP protocol overview
* `docs/Main.md` — UI design & architecture spec
* `docs/Protocol_Integration.md` — fcproto-aligned notes

## Third-party

* Qt 6 — LGPL
* OpenSSL — Apache 2.0, used indirectly via Qt Network
* Catppuccin color palette — MIT

## Known issues / TODO

* Auth dialog closes on send without waiting for server ack; `authSucceeded` signal is declared but unused
* Credentials sent in clear text over TCP; TLS and hashing needed for production
* Avatar upload has no size limit before base64 allocation
* Packet parsing duplicated between `FynClient` and `main.cpp`; emit parsed `CommandType` + payload
* `include/` should be versioned, not git-ignored
* Add reconnect logic, error surfacing, and bounded input

## Contributing

* Keep serialization single-source-of-truth. Avoid `#pragma pack` reliance across compilers.
* Use Qt parent-child ownership; avoid raw owning pointers.
* Add `-Wall -Wextra -Wpedantic` to CMake and treat warnings as errors in CI.
* Run `test_ui_build.sh` before opening PRs.

## License

See repository policy. The project is intended to be released under GPLv3 with OpenSSL exception, matching the FynChat ecosystem.
