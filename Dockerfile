# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/cinder-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/cinder-worker ./cmd/worker

# Final Stage
FROM alpine:latest

# Install Chromium and dependencies
RUN apk add --no-cache \
    chromium \
    ca-certificates \
    tzdata

# Set env for Chromedp to find chromium
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/

WORKDIR /app
COPY --from=builder /app/cinder-api .
COPY --from=builder /app/cinder-worker .

EXPOSE 8080

CMD ["./cinder-api"]
