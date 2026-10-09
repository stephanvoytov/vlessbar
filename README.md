# VLessBar

A minimal VLESS client for **macOS 10.13 (High Sierra, Intel)** and
**Windows 7+** — a drop-in replacement for the Happ client, with subscription
support and Happ-compatible HWID reporting.

> Target: old systems (High Sierra x86_64, Windows 7). Neither can run anything
> built with Go 1.21+, so the whole toolchain is pinned to **Go 1.20** and the
> bundled Xray core is pinned to **v1.8.3**.

## How it works

- `engine/` — the whole client, pure Go stdlib, **no external dependencies**.
  It parses `vless://` links, fetches subscriptions (with Happ request headers),
  renders an Xray config, launches `xray` as a child process, and toggles the
  macOS system proxy via `networksetup`.
- `resources/xray` — Xray-core v1.8.3, built with Go 1.20 (REALITY support,
  still runnable on 10.13).
- `app/ui.applescript` — the GUI (simple, white system dialogs).
- The shipped bundle is `VLessBar.app`:
  - `Contents/MacOS/VLessBar` — the engine binary (real Mach-O executable)
  - `Contents/Resources/xray` — the Xray binary
  - `Contents/Resources/ui.applescript` — the GUI

Running `VLessBar.app` with no arguments starts the GUI, which drives the same
binary through its CLI commands (`up`, `down`, `list`, `set`, `sub-add`, ...).

## Layout

```
vlessbar/
  engine/           Go source (go.mod, main.go, vless.go, sub.go, xray.go, ...)
  app/              Info.plist, ui.applescript
  resources/xray    Xray-core v1.8.3 (darwin/amd64)   [built, git-ignored]
  make-app.ps1      build engine + assemble VLessBar.app + zip
  make_zip.py       mode-preserving zip helper (sets Unix permissions)
  build-xray.ps1    clone + build Xray-core v1.8.3 with Go 1.20
  make-dmg.sh       build a drag-install .dmg (run on macOS)
  dist/             build output (git-ignored)
```

## Build (on Windows)

Prereqs: Go 1.20 (`go1.20` via golang.org/dl), Python 3, `git`.

```powershell
# 1. Build Xray-core (only needed once, or to refresh)
powershell -ExecutionPolicy Bypass -File .\build-xray.ps1

# 2. Build engine + assemble the .app + zip
powershell -ExecutionPolicy Bypass -File .\make-app.ps1
```

Output: `dist/VLessBar.app` and `dist/VLessBar-macos10.13.zip`.

The zip is written by `make_zip.py`, which stores Unix permissions
(`create_system=3` + `external_attr`), so macOS restores the executable bit on
unpack — **the user never needs `chmod`**.

## Build on macOS (or in CI)

`build-macos.sh` does the whole thing on a Mac — engine + Xray + `.app` + zip +
`.dmg` — using `ditto`/`hdiutil`, so permissions are native and no fixups are
needed:

```bash
GO=/path/to/go1.20 ./build-macos.sh
```

GitHub Actions (`.github/workflows/build.yml`) runs the same script on a
`macos-latest` runner, uploads the zip and dmg as artifacts, and publishes them
as a release on `v*` tags.

## Install (on the Mac)

1. Unpack `VLessBar-macos10.13.zip` → `VLessBar.app` (exec bits already set).
2. Launch it. It offers to install itself into `Applications`
   (falls back to `~/Applications` when not admin, no password needed).
3. If it was downloaded from the internet, macOS may show "unidentified
   developer" once: right-click → Open → Open. (Files from AirDrop/USB are not
   quarantined and just open.)

Then: **Подписка** → paste your `https://` subscription link, pick a server,
**Подключить**. The proxy is set via `networksetup`
(HTTP `127.0.0.1:10809`, SOCKS `127.0.0.1:10808`).

## Happ compatibility

Requests send the Happ device/profile headers:

- `User-Agent: Happ/3.13.0`
- `x-hwid` — a stable, once-generated UUID
- `x-device-os: macOS`, `x-ver-os`, `x-device-model`, `x-device-locale`

Responses are interpreted by headers, not status code (the panel returns
`200` even on refusal): `x-hwid-not-supported`, `x-hwid-max-devices-reached`,
`subscription-userinfo`, `profile-web-page-url`, `announce`.

Encrypted `happ://crypto` subscriptions are intentionally out of scope — plain
`https://` links only.

## State

`~/.vlessbar/` — `state.json`, `config.json`, `xray.log`, `xray.pid`.

## Packaging notes

- **DMG**: `make-dmg.sh` builds a standard drag-install disk image, but it must
  run on macOS (`hdiutil` is macOS-only). A `.dmg` is a Disk Image container
  over HFS+/APFS, so it preserves Unix permissions natively and is the
  conventional way to distribute a GUI app on macOS.
- **Signing**: without an Apple Developer ID the app is unsigned, so Gatekeeper
  requires a one-time right-click → Open. Signing/notarization needs macOS and
  a paid certificate.

## Windows (7 / 8 / 8.1 / 10 / 11)

`VLessBar.exe` is a native Win32 GUI (stdlib only, no cgo, no Electron).
Bundled `xray.exe` is Xray-core v1.8.3 so it still runs on Windows 7.

```powershell
# build the engine (Go 1.20)
cd engine; $env:GOOS='windows'; $env:GOARCH='amd64'; go build -o ..\dist-win\VLessBar.exe .
```

`dist-win/` needs `VLessBar.exe` + `xray.exe` next to each other. CI builds
`VLessBar-windows-x64.zip` on every `v*` tag.

### What the UI does

- **server list** — double-click selects a server; `>` marks the active one
- **Обновить подписку / Ввести ссылку** — `https://` subscription, manual
  `vless://` links survive later subscription refreshes
- **Мой IP** — public IP directly, exit IP through the tunnel, TCP RTT to the server
- **Пинг** — latency measured through the tunnel (3 requests, averaged)
- **Подключить/Отключить** — one toggle button that follows the tunnel state
- **Настройки** — app version, Xray core version, LAN gateway toggle, and app
  self-update (GitHub releases; replaces only `VLessBar.exe`, never the core)

Status line shows: tunnel state, active server, its address, and your public IP.

### LAN gateway

Settings → "Разрешить LAN" binds the local proxies on `0.0.0.0` instead of
loopback, so other devices on the network can use `http://<pc-ip>:10809`
(HTTP) or `<pc-ip>:10808` (SOCKS). Windows Firewall must allow inbound
connections on those ports.

CLI equivalent: `VLessBar.exe lan on|off`.

### CLI

`version`, `help`, `hwid`, `sub-add <url>`, `add <vless://...>`, `sub-update`,
`list`, `set <idx>`, `up`, `down`, `lan [on|off]`, `check-update`, `update`,
`conn`, `ping`, `ping-tunnel`, `ip`, `status`, `gui-state`, `gui-servers`.
