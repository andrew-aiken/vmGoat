#!/bin/bash

# Script to download the appropriate VMGoat binary based on OS and architecture
set -e

GITHUB_REPO="andrew-aiken/vmGoat"
RELEASE_TAG="latest"
BINARY_NAME="vmGoat"

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        *)          echo "unknown" ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              echo "unknown" ;;
    esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ "$OS" = "unknown" ] || [ "$ARCH" = "unknown" ]; then
    echo "Error: Unsupported operating system or architecture."
    echo "OS: $(uname -s), Architecture: $(uname -m)"
    exit 1
fi

DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BINARY_NAME}-${OS}-${ARCH}"

echo "Downloading VMGoat binary for ${OS}-${ARCH}..."

curl -s -L -o "${BINARY_NAME}" "${DOWNLOAD_URL}" 2>/dev/null

chmod +x "${BINARY_NAME}"

echo "Installation complete! Run './vmGoat --help' to get started."
