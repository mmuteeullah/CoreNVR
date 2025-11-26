# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o corenvr ./cmd/corenvr

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    tzdata \
    ca-certificates

# Create non-root user
RUN addgroup -S corenvr && adduser -S corenvr -G corenvr

# Create directories
RUN mkdir -p /etc/corenvr /var/log/corenvr /recordings && \
    chown -R corenvr:corenvr /var/log/corenvr /recordings

# Copy binary from builder
COPY --from=builder /app/corenvr /usr/local/bin/corenvr

# Copy example config
COPY configs/config.example.yaml /etc/corenvr/config.example.yaml

# Set permissions
RUN chmod +x /usr/local/bin/corenvr

# Volume for recordings and config
VOLUME ["/recordings", "/etc/corenvr"]

# Expose web UI port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run as non-root user
USER corenvr

# Default command
ENTRYPOINT ["/usr/local/bin/corenvr"]
CMD ["-config", "/etc/corenvr/config.yaml"]
