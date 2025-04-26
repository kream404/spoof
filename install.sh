#!/bin/bash

set -e

detect_shell_rc() {
    if [ -n "$ZSH_VERSION" ]; then
        echo "$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        echo "$HOME/.bashrc"
    else
        if [ -f "$HOME/.zshrc" ]; then
            echo "$HOME/.zshrc"
        else
            echo "$HOME/.bashrc"
        fi
    fi
}

GO_VERSION="1.24.2"
OS="$(uname -s)"
ARCH="$(uname -m)"

if [ "$OS" = "Linux" ]; then
    GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
elif [ "$OS" = "Darwin" ]; then
    if [ "$ARCH" = "x86_64" ]; then
        GO_TARBALL="go${GO_VERSION}.darwin-amd64.tar.gz"
    elif [ "$ARCH" = "arm64" ]; then
        GO_TARBALL="go${GO_VERSION}.darwin-arm64.tar.gz"
    else
        echo "Unsupported Mac architecture: $ARCH"
        exit 1
    fi
else
    echo "Unsupported OS: $OS"
    exit 1
fi

SHELL_RC=$(detect_shell_rc)
PATH_UPDATE_NEEDED=false

# Install Go if missing
if ! command -v go &> /dev/null; then
    echo "🔵 Installing Go ($GO_TARBALL)..."
    curl -O "https://go.dev/dl/${GO_TARBALL}"

    rm -rf "$HOME/go"
    mkdir -p "$HOME/go"
    tar -C "$HOME" -xzf "${GO_TARBALL}"

    if ! grep -q "$HOME/go/bin" "$SHELL_RC"; then
        echo 'export PATH=$PATH:$HOME/go/bin' >> "$SHELL_RC"
        PATH_UPDATE_NEEDED=true
    fi

    export PATH=$PATH:$HOME/go/bin
    echo "Go installed to $HOME/go and added to PATH."
else
    echo "Go already installed. Skipping installation."
fi

# Clone spoof repo if missing
if [ ! -d "spoof" ]; then
    echo "🔵 Cloning spoof repository..."
    git clone https://github.com/kream404/spoof.git
else
    echo "🔵 Repo already exists. Pulling latest changes..."
    cd spoof
    git pull
    cd ..
fi

cd spoof
echo "🔵 Building spoof CLI..."
go build -o spoof main.go

if [ ! -d "$HOME/go/bin" ]; then
    mkdir -p "$HOME/go/bin"
fi

mv -f spoof "$HOME/go/bin/"

if ! grep -q "$HOME/go/bin" "$SHELL_RC"; then
    echo 'export PATH=$PATH:$HOME/go/bin' >> "$SHELL_RC"
    PATH_UPDATE_NEEDED=true
fi

if [ "$PATH_UPDATE_NEEDED" = true ]; then
    echo "🔵 Updated your $SHELL_RC file."
    echo "Please restart your terminal or run: source $SHELL_RC"
fi

echo "Install complete!"
echo "---------------------"
echo "You can now run: 'spoof -v' to verify your install."
echo "Usage: spoof -c /path/to/config.json"
