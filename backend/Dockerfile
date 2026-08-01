# Stage 1: Build
FROM golang:1.22-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /remotehunter ./cmd/server/main.go

# Stage 2: Runtime with Chromium for headless scraping
FROM alpine:3.19

# Install Chromium + runtime deps
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    chromium \
    chromium-chromedriver \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    font-noto-emoji \
    udev

# Tell chromedp where to find Chromium on Alpine
ENV CHROME_PATH=/usr/bin/chromium-browser
# Chromium flags for Docker (no sandbox needed inside container)
ENV CHROMIUM_FLAGS="--no-sandbox --disable-setuid-sandbox --disable-dev-shm-usage --disable-gpu"

WORKDIR /app
COPY --from=builder /remotehunter .

EXPOSE 8080

ENTRYPOINT ["./remotehunter"]
