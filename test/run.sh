#!/bin/bash
set -e

# Directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
DOCKER_DIR="$SCRIPT_DIR/docker"

# Default values
DOTFILES_URL=""
NO_INSTALL=false
NO_EXAMPLES=false

# Load .sandbox.env if it exists
if [ -f "$ROOT_DIR/.sandbox.env" ]; then
    echo "Loading configuration from .sandbox.env"
    source "$ROOT_DIR/.sandbox.env"
fi

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --url)
            DOTFILES_URL="$2"
            shift 2
            ;;
        --no-install)
            NO_INSTALL=true
            shift
            ;;
        --no-examples)
            NO_EXAMPLES=true
            shift
            ;;
        *)
            echo "Unknown argument: $1"
            exit 1
            ;;
    esac
done

echo "Building g4d Linux binary..."
cd "$ROOT_DIR"
GOOS=linux GOARCH=amd64 go build -o bin/g4d-linux-amd64 ./cmd/g4d

# Copy assets to docker context
cp bin/g4d-linux-amd64 "$DOCKER_DIR/g4d"
cp scripts/install.sh "$DOCKER_DIR/install.sh"
cp -r examples "$DOCKER_DIR/examples"

# Detect container runtime
if command -v docker &> /dev/null && docker info &> /dev/null; then
    RUNTIME="docker"
elif command -v podman &> /dev/null && podman info &> /dev/null; then
    RUNTIME="podman"
else
    echo "Error: No working container runtime found."
    exit 1
fi

echo "Using container runtime: $RUNTIME"
echo "Building Docker image..."
$RUNTIME build -t g4d-sandbox \
    --build-arg DOTFILES_URL="$DOTFILES_URL" \
    --build-arg NO_INSTALL="$NO_INSTALL" \
    --build-arg NO_EXAMPLES="$NO_EXAMPLES" \
    "$DOCKER_DIR"

# Cleanup
rm "$DOCKER_DIR/g4d"
rm "$DOCKER_DIR/install.sh"
rm -rf "$DOCKER_DIR/examples"

echo "Starting sandbox..."
$RUNTIME run -it --rm \
    -e TERM=xterm-256color \
    g4d-sandbox
