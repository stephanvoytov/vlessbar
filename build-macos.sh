#!/bin/bash
# Build VLessBar for macOS 10.13 (High Sierra, Intel).
# Runs on macOS (locally and in GitHub Actions): produces
#   dist/VLessBar.app, dist/VLessBar-macos10.13.zip, dist/VLessBar.dmg
#
# Uses Go 1.20 so the binaries stay runnable on 10.13 (Go 1.21+ drops it).
# Override the toolchain with:  GO=/path/to/go1.20 ./build-macos.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
GO="${GO:-go}"
XRAY_VER="${XRAY_VER:-v1.8.3}"

DIST="$ROOT/dist"
APP="$DIST/VLessBar.app"
CONTENTS="$APP/Contents"
MACOS="$CONTENTS/MacOS"
RES="$CONTENTS/Resources"
XRAY_BIN="$ROOT/resources/xray"

echo "==> toolchain: $("$GO" version)"

if [ ! -f "$XRAY_BIN" ]; then
	echo "==> building Xray-core $XRAY_VER"
	WORK="$(mktemp -d)"
	trap 'rm -rf "$WORK"' EXIT
	git clone --depth 1 --branch "$XRAY_VER" https://github.com/XTLS/Xray-core.git "$WORK/xray"
	( cd "$WORK/xray/main" && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 "$GO" build -trimpath -ldflags "-s -w" -o "$XRAY_BIN" . )
else
	echo "==> reusing $XRAY_BIN"
fi

echo "==> building engine"
rm -rf "$DIST"
mkdir -p "$MACOS" "$RES"
( cd "$ROOT/engine" && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 "$GO" build -trimpath -ldflags "-s -w" -o "$MACOS/VLessBar" . )

echo "==> assembling bundle"
cp "$ROOT/app/Info.plist" "$CONTENTS/Info.plist"
cp "$ROOT/app/ui.applescript" "$RES/ui.applescript"
cp "$XRAY_BIN" "$RES/xray"
chmod +x "$MACOS/VLessBar" "$RES/xray"

echo "==> zip"
ZIP="$DIST/VLessBar-macos10.13.zip"
( cd "$DIST" && ditto -c -k --sequesterRsrc --keepParent "VLessBar.app" "$ZIP" )

echo "==> dmg"
DMG="$DIST/VLessBar.dmg"
STAGE="$(mktemp -d)"
cp -R "$APP" "$STAGE/"
ln -s /Applications "$STAGE/Applications"
rm -f "$DMG"
hdiutil create -volname "VLessBar" -srcfolder "$STAGE" -ov -format UDZO "$DMG"
rm -rf "$STAGE"

echo "==> done"
ls -lh "$ZIP" "$DMG"
