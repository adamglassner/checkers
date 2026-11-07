#!/usr/bin/env bash
# Builds Checkers for Windows (64-bit).
# Output: dist/Games/Checkers/ with Checkers.exe and its text files,
# ready for the Ironite Launcher (the release zip) or for
# installer/CheckersSetup.iss.
# Needs Go 1.24+ and bash (Git Bash on Windows is fine). No C compiler needed.
set -euo pipefail
cd "$(dirname "$0")"
ROOT=$PWD
BUNDLE=Checkers
EXE=Checkers
OUT=$ROOT/dist/Games/$BUNDLE
export GOFLAGS=-mod=mod
rm -rf "$ROOT/dist" && mkdir -p "$OUT"

VERSION=$(grep -m1 -oE '^## \[[0-9.]+' CHANGELOG.md | tr -d '#[ ')
CODE_VERSION=$(grep -m1 -oE 'const Version = "[0-9.]+"' game.go | grep -oE '[0-9.]+')
ISS_VERSION=$(grep -m1 -oE 'GameVersion "[0-9.]+"' installer/CheckersSetup.iss | grep -oE '[0-9.]+')
[ "$CODE_VERSION" = "$VERSION" ] || { echo "game.go Version ($CODE_VERSION) doesn't match CHANGELOG.md ($VERSION)"; exit 1; }
[ "$ISS_VERSION" = "$VERSION" ] || { echo "installer/CheckersSetup.iss ($ISS_VERSION) doesn't match CHANGELOG.md ($VERSION)"; exit 1; }

# Windows resource (icon + version info) for the .exe
( cd tools/mkres && GOFLAGS=-mod=vendor go build -o "$ROOT/mkres.tmp" . )
"$ROOT/mkres.tmp" -version "$VERSION" -icon assets/icon-1024.png -out rsrc_windows_amd64.syso \
  -name "Checkers" -exe "$EXE.exe" -internal "$EXE" -identity "Ironite.$BUNDLE" -desc "Checkers - Hollow Creek General Store"
rm -f "$ROOT/mkres.tmp"

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-H windowsgui -s -w" -o "$OUT/$EXE.exe" .
rm -f rsrc_windows_amd64.syso

crlf() { sed 's/\r$//; s/$/\r/' "$1" > "$2"; }
crlf CHANGELOG.md "$OUT/CHANGELOG.txt"
crlf assets/fonts/OFL.txt "$OUT/OFL.txt"
sed "s/@VERSION@/$VERSION/g" README.txt  | sed 's/\r$//; s/$/\r/' > "$OUT/README.txt"
sed "s/@VERSION@/$VERSION/g" CREDITS.txt | sed 's/\r$//; s/$/\r/' > "$OUT/CREDITS.txt"
printf '%s\r\n' "$VERSION" > "$OUT/VERSION"
echo "Checkers $VERSION -> $OUT/$EXE.exe"
