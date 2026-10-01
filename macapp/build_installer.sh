#!/bin/bash
# Rebuilds "IPTV Player.app" (universal arm64+x86_64 binary, self-contained
# -- no Ruby/Go/Node needed on the machine that runs it) from the current
# project source and packages it into "IPTV Player.pkg", a standard macOS
# installer package.
#
# Run this after `cd ../frontend && npm run build` whenever the web app
# changes and you want a fresh installer.
set -euo pipefail
cd "$(dirname "$0")"

GO="../.gotoolchain/go/bin/go"
APP_BUNDLE="IPTV Player.app"
PKG_NAME="IPTV Player.pkg"

if [ ! -x "$GO" ]; then
  echo "Go toolchain not found at $GO -- see ../.gotoolchain/README.txt" >&2
  exit 1
fi
if [ ! -d "../public/assets" ]; then
  echo "../public looks empty -- run 'cd ../frontend && npm run build' first." >&2
  exit 1
fi

echo "Building universal (arm64 + x86_64) binary..."
( cd ../goserver
  GOOS=darwin GOARCH=arm64 "$(cd .. && pwd)/.gotoolchain/go/bin/go" build -ldflags="-s -w" -o /tmp/iptvplayer_arm64 .
  GOOS=darwin GOARCH=amd64 "$(cd .. && pwd)/.gotoolchain/go/bin/go" build -ldflags="-s -w" -o /tmp/iptvplayer_amd64 .
)
lipo -create -output /tmp/iptvplayer_universal /tmp/iptvplayer_arm64 /tmp/iptvplayer_amd64
rm -f /tmp/iptvplayer_arm64 /tmp/iptvplayer_amd64

echo "Refreshing bundle contents..."
rm -rf "$APP_BUNDLE/Contents/Resources/app"
mkdir -p "$APP_BUNDLE/Contents/Resources/app"
cp /tmp/iptvplayer_universal "$APP_BUNDLE/Contents/Resources/app/iptvplayer"
chmod +x "$APP_BUNDLE/Contents/Resources/app/iptvplayer"
cp -R ../public "$APP_BUNDLE/Contents/Resources/app/"
rm -f /tmp/iptvplayer_universal

# Ad-hoc signed (no Apple Developer account/notarization here) -- gives the
# bundle a valid integrity signature, but macOS Gatekeeper will still flag
# it as being from an "unidentified developer" if launched by double-click.
# See README.md for why the .pkg + `sudo installer` CLI install path
# sidesteps that entirely, which is the recommended install method.
codesign --force --deep --sign - "$APP_BUNDLE"

echo "Building installer package..."
rm -rf pkgroot
mkdir -p pkgroot
cp -R "$APP_BUNDLE" pkgroot/
rm -f "$PKG_NAME"
pkgbuild --root pkgroot \
  --identifier com.oron.iptvplayer.pkg \
  --version 1.0 \
  --install-location /Applications \
  "$PKG_NAME"
rm -rf pkgroot

echo "Built $(pwd)/$PKG_NAME"
