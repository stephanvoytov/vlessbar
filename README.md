# VLessBar

A minimal VLESS client for **macOS 10.13 (High Sierra, Intel)** — built as a
drop-in replacement for the Happ client, with subscription support and
Happ-compatible HWID reporting.

> Target: High Sierra x86_64 only. High Sierra cannot run anything built with
> Go 1.21+, so the whole toolchain is pinned to **Go 1.20**.

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
