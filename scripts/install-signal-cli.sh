#!/bin/bash
set -e

VERSION="0.13.23"
INSTALL_DIR="$(pwd)/signal-cli"

echo "Installing signal-cli $VERSION..."

if [ -d "$INSTALL_DIR" ]; then
    echo "signal-cli already installed at $INSTALL_DIR"
    exit 0
fi

TARBALL="signal-cli-${VERSION}.tar.gz"
URL="https://github.com/AsamK/signal-cli/releases/download/v${VERSION}/${TARBALL}"

echo "Downloading from $URL..."
curl -L -o "/tmp/$TARBALL" "$URL"

echo "Extracting..."
tar -xzf "/tmp/$TARBALL" -C "$(pwd)"
mv "signal-cli-${VERSION}" "$INSTALL_DIR"

rm "/tmp/$TARBALL"

echo "✓ signal-cli installed to $INSTALL_DIR"
echo "Add to PATH: export PATH=\"$INSTALL_DIR/bin:\$PATH\""
