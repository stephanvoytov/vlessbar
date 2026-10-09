#!/bin/bash
# Build a drag-to-install DMG. RUN THIS ON macOS (hdiutil is macOS-only).
#
# Usage:  ./make-dmg.sh [app] [out]
#   ./make-dmg.sh                       # dist/VLessBar.app -> dist/VLessBar.dmg
#
# Produces a compressed disk image containing VLessBar.app and an
# "Applications" symlink, so the user drags the app onto Applications.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
APP="${1:-$ROOT/dist/VLessBar.app}"
OUT="${2:-$ROOT/dist/VLessBar.dmg}"
VOL="VLessBar"

if [ ! -d "$APP" ]; then
	echo "app not found: $APP" >&2
	exit 1
fi

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

cp -R "$APP" "$STAGE/"
ln -s /Applications "$STAGE/Applications"

rm -f "$OUT"
hdiutil create -volname "$VOL" -srcfolder "$STAGE" -ov -format UDZO "$OUT"
echo "wrote $OUT"
