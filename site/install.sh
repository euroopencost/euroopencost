#!/bin/sh
set -e

REPO="euroopencost/euroopencost"
BINARY="eucost"
INSTALL_DIR="/usr/local/bin"

# OS und Arch erkennen
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: $OS"
    echo "Download manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

# Neueste Version von GitHub API
VERSION="$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version. Check https://github.com/$REPO/releases"
  exit 1
fi

FILENAME="${BINARY}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo "Installing eucost ${VERSION} (${OS}/${ARCH})..."
curl -sSL "$URL" -o "/tmp/${BINARY}"
chmod +x "/tmp/${BINARY}"

if [ -w "$INSTALL_DIR" ]; then
  mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "Installed to: ${INSTALL_DIR}/${BINARY}"
echo "Run: eucost plan <terraform-plan.json>"
