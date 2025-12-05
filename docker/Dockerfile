# Waterflow Dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o waterflow \
    ./cmd/waterflow

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S waterflow && \
    adduser -u 1001 -S waterflow -G waterflow

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/waterflow .

# Change ownership
RUN chown waterflow:waterflow waterflow

# Switch to non-root user
USER waterflow

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./waterflow version || exit 1

# Run the binary
ENTRYPOINT ["./waterflow"]
CMD ["server"]