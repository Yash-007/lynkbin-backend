#!/bin/bash
set -e

# Install Playwright browsers
echo "Installing Playwright browsers..."
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium

# Build the Go application
echo "Building Go application..."
go build -o bin/main cmd/main.go

echo "Build completed successfully!"
