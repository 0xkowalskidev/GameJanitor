#!/bin/bash
set -e

REPO="warsmite/gamejanitor"
INSTALL_DIR="/usr/local/bin"
BINARY="gamejanitor"

echo "Installing gamejanitor..."

# Require root
if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: This script must be run as root."
    echo "  curl -fsSL https://get.gamejanitor.com | sudo sh"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "ERROR: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ]; then
    echo "ERROR: Unsupported OS: $OS (only Linux is supported)"
    exit 1
fi

# Download latest release
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}-${OS}-${ARCH}"
echo "Downloading ${BINARY} (${OS}/${ARCH})..."

TMP=$(mktemp)
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP"; then
    echo "ERROR: Failed to download from ${DOWNLOAD_URL}"
    echo "Check https://github.com/${REPO}/releases for available releases."
    rm -f "$TMP"
    exit 1
fi

chmod +x "$TMP"
mv "$TMP" "${INSTALL_DIR}/${BINARY}"
echo "Binary installed to ${INSTALL_DIR}/${BINARY}"

# Detect runtime
RUNTIME="process"
if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    RUNTIME="docker"
    echo "Detected Docker — using container runtime."
else
    echo "No Docker detected — using sandbox runtime (bwrap)."
    # Install bwrap if missing
    if ! command -v bwrap &>/dev/null; then
        echo "Installing bubblewrap..."
        if command -v apt-get &>/dev/null; then
            apt-get install -y -qq bubblewrap
        elif command -v dnf &>/dev/null; then
            dnf install -y -q bubblewrap
        elif command -v pacman &>/dev/null; then
            pacman -S --noconfirm bubblewrap
        elif command -v zypper &>/dev/null; then
            zypper install -y bubblewrap
        else
            echo "WARNING: Could not install bubblewrap automatically."
            echo "Install it manually: https://github.com/containers/bubblewrap"
        fi
    fi
fi

# Install systemd service
echo "Setting up systemd service..."
gamejanitor install --runtime "$RUNTIME"

echo ""
echo "Gamejanitor is installed and running."
echo ""
echo "  Web UI:   http://localhost:8080"
echo "  Status:   systemctl status gamejanitor"
echo "  Logs:     journalctl -u gamejanitor -f"
echo ""
echo "Get started:"
echo "  gamejanitor create \"My Server\" minecraft-java --env EULA=true"
