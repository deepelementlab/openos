# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S aos && \
    adduser -u 1000 -S aos -G aos

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always 2>/dev/null || echo 'dev')" \
    -o /app/bin/aos ./cmd/aos

# Test stage (optional)
FROM builder AS tester
RUN go test ./...

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 -S aos && \
    adduser -u 1000 -S aos -G aos

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/aos /usr/local/bin/aos
COPY --from=builder /app/configs /app/configs

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Create necessary directories
RUN mkdir -p /app/logs /app/data && \
    chown -R aos:aos /app

# Set environment variables
ENV PATH="/usr/local/bin:${PATH}" \
    AOS_CONFIG_PATH="/app/configs/config.yaml"

# Switch to non-root user
USER aos

# Expose ports (default API port)
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Entrypoint
ENTRYPOINT ["aos"]

# Default command (pass config path)
CMD ["--config", "/app/configs/config.yaml"]