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
        echo "❌ Unsupported Mac architecture: $ARCH"
        exit 1
    fi
else
    echo "❌ Unsupported OS: $OS"
    exit 1
fi

#install go
if ! command -v go &> /dev/null; then
    echo "🔵 Installing Go ($GO_TARBALL)..."

    wget "https://go.dev/dl/${GO_TARBALL}"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "${GO_TARBALL}"

    SHELL_RC=$(detect_shell_rc)

    echo 'export PATH=$PATH:/usr/local/go/bin' >> "$SHELL_RC"
    export PATH=$PATH:/usr/local/go/bin

    echo "Go installed."
    echo "Added Go to your PATH in $SHELL_RC."
    echo "Please restart your terminal or run: source $SHELL_RC"
else
    echo "Go already installed. Skipping installation."
fi

# clone and build
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

# move spoof binary to $HOME/go/bin (or /usr/local/bin)
if [ ! -d "$HOME/go/bin" ]; then
    mkdir -p "$HOME/go/bin"
fi

mv spoof "$HOME/go/bin/"

# Add $HOME/go/bin to PATH if not already there
SHELL_RC=$(detect_shell_rc)

if ! grep -q "$HOME/go/bin" "$SHELL_RC"; then
    echo "export PATH=\$PATH:$HOME/go/bin" >> "$SHELL_RC"
    echo "Added $HOME/go/bin to your PATH in $SHELL_RC"
    echo "Please restart your terminal or run: source $SHELL_RC"
fi

echo "Done! You can now run spoof -c /path/to/config.json"

echo "Done! You can now run ./spoof -c /path/to/config.json"
