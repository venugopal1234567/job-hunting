# Stage 1: Build
FROM golang:1.22-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /remotehunter ./cmd/server/main.go

# Stage 2: Runtime with Google Chrome for headless scraping
FROM debian:12-slim

# Install system dependencies and Google Chrome
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    gnupg \
    procps \
    poppler-utils \
    && curl -fsSL https://dl-ssl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /usr/share/keyrings/googlechrome-keyring.gpg \
    && echo "deb [arch=amd64 signed-by=/usr/share/keyrings/googlechrome-keyring.gpg] http://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google-chrome.list \
    && apt-get update && apt-get install -y --no-install-recommends \
    google-chrome-stable \
    && rm -rf /var/lib/apt/lists/*

# Tell chromedp where to find Google Chrome
ENV CHROME_PATH=/usr/bin/google-chrome-stable
# Chrome flags for Docker (no sandbox needed inside container)
ENV CHROMIUM_FLAGS="--no-sandbox --disable-setuid-sandbox --disable-dev-shm-usage --disable-gpu"

WORKDIR /app
COPY --from=builder /remotehunter .

EXPOSE 8080

ENTRYPOINT ["./remotehunter"]
