#!/usr/bin/env bash

set -e

echo "Building bbkt..."
# Build the binary
go build -ldflags="-s -w" -o bbkt ./cmd/bbkt

# Determine destination directory
DEST_DIR="$HOME/.local/bin"

if [ ! -d "$DEST_DIR" ]; then
    echo "Creating $DEST_DIR..."
    mkdir -p "$DEST_DIR"
fi

echo "Installing bbkt to $DEST_DIR..."
mv bbkt "$DEST_DIR/"

echo "Installation complete!"
echo "Ensure that $DEST_DIR is in your system PATH using:"
echo '  export PATH="$HOME/.local/bin:$PATH"'
