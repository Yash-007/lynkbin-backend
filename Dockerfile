# Use official Golang image with Debian
FROM golang:1.23-bookworm

# Install Chromium dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    wget \
    gnupg \
    libnss3 \
    libnspr4 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libdbus-1-3 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libpango-1.0-0 \
    libcairo2 \
    libasound2 \
    libatspi2.0-0 \
    libx11-6 \
    libx11-xcb1 \
    libxcb1 \
    libxext6 \
    libxshmfence1 \
    fonts-liberation \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install Playwright browsers with dependencies
RUN go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium

# Build the application
RUN go build -o main cmd/main.go

# Set environment variables
ENV PLAYWRIGHT_BROWSERS_PATH=/root/.cache/ms-playwright
ENV HOME=/root

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
