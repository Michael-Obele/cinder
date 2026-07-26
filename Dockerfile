# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/cinder-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/cinder-worker ./cmd/worker

# Final Stage
FROM alpine:latest

# Install Chromium, fonts, runtime dependencies, and tini for zombie reaping
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    tzdata \
    ttf-freefont \
    font-noto-emoji \
    wqy-zenhei \
    tini

# Set env for Chromedp to find chromium
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/

WORKDIR /app
COPY --from=builder /app/cinder-api .
COPY --from=builder /app/cinder-worker .
COPY --from=builder /app/internal/api/docs ./internal/api/docs

# Create a symlink for chromedp compatibility
RUN ln -sf /usr/bin/chromium-browser /usr/bin/google-chrome-stable

EXPOSE 8080

# tini reaps zombie processes that Chrome spawns
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./cinder-api"]
