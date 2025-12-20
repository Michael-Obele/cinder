#!/bin/bash
set -e

# Install dependencies
echo "INFO: Installing common fonts and libraries..."
apt-get update
apt-get install -y \
    fonts-ipafont-gothic \
    fonts-wqy-zenhei \
    fonts-thai-tlwg \
    fonts-kacst \
    fonts-freefont-ttf \
    libxss1 \
    wget \
    gnupg \
    --no-install-recommends

# Install Chrome/Chromium based on arch
ARCH=$(dpkg --print-architecture)
echo "INFO: Detected architecture: $ARCH"

if [ "$ARCH" = "amd64" ]; then
    echo "INFO: Installing Google Chrome for amd64..."
    wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add -
    echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google.list
    apt-get update
    apt-get install -y google-chrome-stable --no-install-recommends
    BROWSER_EXEC="google-chrome-stable"
elif [ "$ARCH" = "arm64" ]; then
    echo "INFO: Installing Chromium for arm64..."
    apt-get install -y chromium --no-install-recommends
    BROWSER_EXEC="chromium"
else
    echo "ERROR: Unsupported architecture: $ARCH" >&2
    exit 1
fi

# Clean up
rm -rf /var/lib/apt/lists/*

# Move executable to local dir
chrome_path=$(which "$BROWSER_EXEC")
if [ -n "$chrome_path" ]; then
    mv "$chrome_path" ./google-chrome-stable
    echo "INFO: Moved executable to ./google-chrome-stable"
else
    echo "ERROR: Browser executable '$BROWSER_EXEC' not found." >&2
    exit 1
fi
