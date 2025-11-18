# Multi-stage Dockerfile for URL Crawler v10
# ==========================================

# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X 'main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)' -X 'main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')'" \
    -o /build/url_crawler_twotier \
    .

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 crawler && \
    adduser -D -u 1000 -G crawler crawler

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/url_crawler_twotier /app/url_crawler_twotier

# Create directories for data
RUN mkdir -p /app/downloads /app/logs && \
    chown -R crawler:crawler /app

# Switch to non-root user
USER crawler

# Set environment variables
ENV DOWNLOAD_DIR=/app/downloads \
    LOG_DIR=/app/logs

# Expose any ports if needed (for future API)
# EXPOSE 8080

# Health check (optional - for future use)
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Volume for persistent data
VOLUME ["/app/downloads", "/app/logs"]

# Run the application
ENTRYPOINT ["/app/url_crawler_twotier"]

# Labels
LABEL maintainer="Your Name <your.email@example.com>" \
      description="Multi-NIC URL Crawler v10 - Two-Tier Edition" \
      version="10.0" \
      org.opencontainers.image.source="https://github.com/yourusername/url_crawler_twotier" \
      org.opencontainers.image.licenses="MIT"
