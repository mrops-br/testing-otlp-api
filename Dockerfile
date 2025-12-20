# Stage 1: Build the Go application
FROM golang:1.21-alpine3.19 AS builder

# Install build dependencies with pinned versions
RUN apk add --no-cache \
    git=~2.43 \
    ca-certificates=~20250911 \
    tzdata=~2025b

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to reduce binary size by stripping debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o products-api \
    main.go

# Stage 2: Create minimal runtime image
FROM alpine:3.19

# Install ca-certificates for HTTPS connections and timezone data
RUN apk --no-cache add \
    ca-certificates=~20250911 \
    tzdata=~2025b \
    wget=~1.21

# Create non-root user
RUN addgroup -g 1001 appgroup && \
    adduser -D -u 1001 -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/products-api .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./products-api"]
