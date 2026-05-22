#!/bin/bash
set -e

BUILD_DIR="./build"
APP_NAME="TLO"

VERSION=$(cat version.txt | tr -d '[:space:]')
LDFLAGS="-s -w -X main.Version=${VERSION}"

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

echo "==> Building version ${VERSION} for Linux (amd64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}-${VERSION}_linux_amd64" .

echo "==> Building version ${VERSION} for macOS (amd64)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}-${VERSION}_darwin_amd64" .

echo "==> Building version ${VERSION} for Windows (amd64)..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}-${VERSION}_windows_amd64.exe" .

echo ""
echo "Done. Binaries in $BUILD_DIR:"
ls -lh "$BUILD_DIR"
