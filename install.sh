#!/bin/bash

# Script to download the appropriate VMGoat binary based on OS and architecture
set -e

GITHUB_REPO="andrew-aiken/vmGoat"
RELEASE_TAG="latest"
BINARY_NAME="vmGoat"

# Print colored output
print_message() {
    local color_code="$1"
    local message="$2"
    echo "\033[${color_code}m${message}\033[0m"
}

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
    print_message "31" "Error: Unsupported operating system or architecture."
    print_message "31" "OS: $(uname -s), Architecture: $(uname -m)"
    exit 1
fi

DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BINARY_NAME}-${OS}-${ARCH}"

print_message "36" "Downloading VMGoat binary for ${OS}-${ARCH}..."
if command -v curl >/dev/null 2>&1; then
    curl -s -L -o "${BINARY_NAME}" "${DOWNLOAD_URL}" 2>/dev/null
elif command -v wget >/dev/null 2>&1; then
    wget -q -O "${BINARY_NAME}" "${DOWNLOAD_URL}" 2>/dev/null
else
    print_message "31" "Error: Neither curl nor wget found. Please install one of them and try again."
    exit 1
fi

# Make the binary executable
chmod +x "${BINARY_NAME}"

print_message "36" "Installation complete! Run './vmGoat --help' to get started."
