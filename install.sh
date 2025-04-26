#!/bin/bash

set -e

GO_VERSION="1.24.2"
OS="$(uname -s)"
ARCH="$(uname -m)"
PATH_UPDATE_NEEDED=false

if [ "$OS" = "Linux" ]; then
    GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
elif [ "$OS" = "Darwin" ]; then
    if [ "$ARCH" != "x86_64" ] && [ "$ARCH" != "arm64" ]; then
        echo "Unsupported Mac architecture: $ARCH"
        exit 1
    fi
else
    echo "Unsupported OS: $OS"
    exit 1
fi

if ! command -v go &> /dev/null; then
    if [ "$OS" = "Linux" ]; then
        echo "Installing Go ($GO_TARBALL)..."
        wget "https://go.dev/dl/${GO_TARBALL}"
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "${GO_TARBALL}"
        export PATH=$PATH:/usr/local/go/bin

        for RC in "$HOME/.bashrc" "$HOME/.zshrc"; do
            if [ -f "$RC" ] && ! grep -q "/usr/local/go/bin" "$RC"; then
                echo 'export PATH=$PATH:/usr/local/go/bin' >> "$RC"
                PATH_UPDATE_NEEDED=true
            fi
        done
    elif [ "$OS" = "Darwin" ]; then
        echo "Installing Go using Homebrew..."
        if ! command -v brew &> /dev/null; then
            echo "Homebrew is not installed. Install Homebrew first: https://brew.sh/"
            exit 1
        fi
        brew install go
    fi
    echo "Go installed and added to PATH."
else
    echo "Go already installed. Skipping installation."
fi

if [ ! -d "spoof" ]; then
    echo "Cloning spoof repository..."
    git clone https://github.com/kream404/spoof.git
else
    echo "Repo already exists. Pulling latest changes..."
    cd spoof
    git pull
    cd ..
fi

cd spoof
echo "Building spoof CLI..."
go build -o spoof main.go

if [ ! -d "$HOME/go/bin" ]; then
    mkdir -p "$HOME/go/bin"
fi

mv -f spoof "$HOME/go/bin/"

for RC in "$HOME/.bashrc" "$HOME/.zshrc"; do
    if [ -f "$RC" ] && ! grep -q "$HOME/go/bin" "$RC"; then
        echo 'export PATH=$PATH:$HOME/go/bin' >> "$RC"
        PATH_UPDATE_NEEDED=true
    fi
done

if [ "$PATH_UPDATE_NEEDED" = true ]; then
    echo "Updated your shell configuration files."
    echo "Please restart your terminal or run: source ~/.bashrc or source ~/.zshrc"
fi

echo "Install complete!"
echo "------------------"
echo "You can now run: 'spoof -v' to verify your install."
echo "Usage: spoof -c /path/to/config.json"
