#!/bin/bash
set -e

REPO="nvandessel/go4dot"
BINARY="g4d"
INSTALL_DIR="/usr/local/bin"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}🐹 Installing go4dot...${NC}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)     OS=linux;;
    Darwin*)    OS=darwin;;
    *)          echo -e "${RED}Unsupported OS: $OS${NC}"; exit 1;;
esac

# Detect Arch
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH=amd64;;
    aarch64) ARCH=arm64;;
    arm64)   ARCH=arm64;;
    *)       echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1;;
esac

echo "Detected Platform: $OS/$ARCH"

# Determine Install Dir
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Fetch Latest Version
LATEST_RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
echo "Fetching latest version from GitHub..."

# Note: this requires jq or complex grep/sed if jq is missing.
# We'll use a simple grep approach to avoid dependencies.
DOWNLOAD_URL=$(curl -s $LATEST_RELEASE_URL | grep "browser_download_url" | grep "$OS-$ARCH" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
    echo -e "${RED}Failed to find a release for $OS/$ARCH${NC}"
    echo "Check https://github.com/$REPO/releases for manual installation."
    exit 1
fi

echo "Downloading $DOWNLOAD_URL..."
TMP_DIR=$(mktemp -d)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/release.tar.gz"

# Extract
echo "Extracting..."
tar -xzf "$TMP_DIR/release.tar.gz" -C "$TMP_DIR"

# Install
echo "Installing to $INSTALL_DIR..."
# The binary might be named 'g4d' or 'g4d-linux-amd64' etc.
if [ -f "$TMP_DIR/$BINARY" ]; then
    mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
else
    # Look for any file starting with the binary name that isn't the archive itself
    EXTRACTED_BINARY=$(ls "$TMP_DIR" | grep "^$BINARY" | grep -v "release.tar.gz" | head -n 1)
    if [ -n "$EXTRACTED_BINARY" ]; then
        mv "$TMP_DIR/$EXTRACTED_BINARY" "$INSTALL_DIR/$BINARY"
    else
        echo -e "${RED}Error: Could not find binary '$BINARY' in the extracted files.${NC}"
        ls -l "$TMP_DIR"
        exit 1
    fi
fi
chmod +x "$INSTALL_DIR/$BINARY"

# Cleanup
rm -rf "$TMP_DIR"

echo -e "${GREEN}✅ Installation successful!${NC}"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${RED}Warning: $INSTALL_DIR is not in your PATH.${NC}"
    echo "You may need to add it to your shell profile (e.g., ~/.zshrc or ~/.bashrc):"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

echo "Run '$BINARY --version' to verify."
