#!/bin/bash

echo "Setting up Chrome 148 and ChromeDriver..."

# Install Chrome Beta
wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
sudo sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ beta main" >> /etc/apt/sources.list.d/google-chrome-beta.list'
sudo apt-get update
sudo apt-get install -y google-chrome-beta

# Download ChromeDriver 151
wget https://storage.googleapis.com/chrome-for-testing-public/151.0.7922.173/linux64/chromedriver-linux64.zip
unzip -o chromedriver-linux64.zip
sudo mv chromedriver-linux64/chromedriver /usr/bin/chromedriver-151
sudo chmod +x /usr/bin/chromedriver-151

# Make it the default
sudo ln -sf /usr/bin/chromedriver-151 /usr/bin/chromedriver
# Verify installations
echo "=== Chrome Version ==="
google-chrome-beta --version

echo "=== ChromeDriver Version ==="
chromedriver --version

echo "Setup complete!"