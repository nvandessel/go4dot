#!/bin/bash
set -e

# Directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
DOCKER_DIR="$SCRIPT_DIR/docker"

echo "Building g4d Linux binary..."
cd "$ROOT_DIR"
GOOS=linux GOARCH=amd64 go build -o bin/g4d-linux-amd64 ./cmd/g4d

# Copy binary and examples to docker context for embedding
cp bin/g4d-linux-amd64 "$DOCKER_DIR/g4d"
cp -r examples "$DOCKER_DIR/examples"

# Detect container runtime (checking for active daemon/functionality)
if command -v docker &> /dev/null && docker info &> /dev/null; then
    RUNTIME="docker"
elif command -v podman &> /dev/null && podman info &> /dev/null; then
    RUNTIME="podman"
else
    echo "Error: No working container runtime found."
    echo "Please ensure Docker daemon is running OR Podman is installed and working."
    rm "$DOCKER_DIR/g4d"
    rm -rf "$DOCKER_DIR/examples"
    exit 1
fi

echo "Using container runtime: $RUNTIME"
echo "Building Docker image..."
$RUNTIME build -t g4d-sandbox "$DOCKER_DIR"

# Cleanup binary and examples from docker dir
rm "$DOCKER_DIR/g4d"
rm -rf "$DOCKER_DIR/examples"

echo "Starting sandbox..."
echo "Type 'g4d' to test the binary."
echo "Examples are pre-installed at ~/examples (minimal and advanced)"
echo "Run 'reset-examples' to restore them to original state."

$RUNTIME run -it --rm \
    -e TERM=xterm-256color \
    g4d-sandbox
