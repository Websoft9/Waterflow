# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with static linking
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'docker') -X main.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -o server cmd/server/main.go

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS support
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 waterflow && \
    adduser -D -u 1000 -G waterflow waterflow

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server /app/server

# Copy config example
COPY config.example.yaml /etc/waterflow/config.yaml

# Change ownership
RUN chown -R waterflow:waterflow /app /etc/waterflow

# Switch to non-root user
USER waterflow

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Run server
ENTRYPOINT ["/app/server"]
CMD ["--config", "/etc/waterflow/config.yaml"]
