#!/usr/bin/env bash
# Builds Checkers.exe and the Checkers installer kit.
#   dist/Checkers/                      Checkers.exe + README, CREDITS, CHANGELOG, licence
#   dist/Checkers-Installer-v<ver>/     copy to Windows, run "Build Installer.bat"
# Needs Go 1.24+. Works offline (the one tool dependency is vendored).
# The version comes from the first entry in web/CHANGELOG.md.
set -euo pipefail
cd "$(dirname "$0")"
ROOT=$PWD
VERSION=$(grep -m1 -oP '^## \[\K[0-9.]+' web/CHANGELOG.md)
ISS_VERSION=$(grep -m1 -oP '#define GameVersion\s+"\K[^"]+' installer/CheckersSetup.iss)
[ "$ISS_VERSION" = "$VERSION" ] || { echo "installer/CheckersSetup.iss GameVersion ($ISS_VERSION) != CHANGELOG ($VERSION)"; exit 1; }
TITLE="Checkers"   # must match the page <title> in web/index.html
OUT=$ROOT/dist; B=$ROOT/build
rm -rf "$OUT" "$B" && mkdir -p "$OUT/Checkers" "$B"
crlf() { sed 's/\r$//; s/$/\r/' "$1" > "$2"; }

echo "== Resources (icon + version $VERSION)"
( cd tools/mkres && GOFLAGS=-mod=vendor go build -o "$B/mkres" . )
cp app/*.go app/go.mod "$B/" && cp -r web "$B/web"
"$B/mkres" -version "$VERSION" -icon assets/icon-1024.png -out "$B/rsrc_windows_amd64.syso" \
  -name "$TITLE" -exe Checkers.exe -internal Checkers -identity Ironite.Checkers -desc "Checkers - Hollow Creek General Store"

echo "== Checkers.exe"
( cd "$B" && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
    -ldflags "-H windowsgui -s -w -X main.gameID=Checkers -X \"main.title=$TITLE\" -X main.version=$VERSION" \
    -o "$OUT/Checkers/Checkers.exe" . )
crlf web/CHANGELOG.md "$OUT/Checkers/CHANGELOG.txt"
crlf web/fonts/OFL.txt "$OUT/Checkers/OFL.txt"
for f in README CREDITS; do sed "s/@VERSION@/$VERSION/g" package/$f.txt > "$B/$f.txt" && crlf "$B/$f.txt" "$OUT/Checkers/$f.txt"; done
printf '%s\r\n' "$VERSION" > "$OUT/Checkers/VERSION"

echo "== Installer kit"
KIT=$OUT/Checkers-Installer-v$VERSION
mkdir -p "$KIT/files/Games" "$KIT/installer/assets" "$KIT/Output"
cp -r "$OUT/Checkers" "$KIT/files/Games/"
for f in installer/*.iss; do crlf "$f" "$KIT/installer/$(basename "$f")"; done
cp installer/assets/* "$KIT/installer/assets/"
crlf "installer/Build Installer.bat" "$KIT/Build Installer.bat"
rm -rf "$B"
ls -la "$OUT/Checkers"; echo "Kit: $KIT"
