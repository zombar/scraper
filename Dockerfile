# Multi-stage build for optimal image size
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build both binaries
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o scraper-api ./cmd/api

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

# Create non-root user
RUN addgroup -g 1000 scraper && \
    adduser -D -u 1000 -G scraper scraper

# Create necessary directories
RUN mkdir -p /app/data && \
    chown -R scraper:scraper /app

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/scraper-api .

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Switch to non-root user
USER scraper

# Create volume for persistent data
VOLUME /app/data

# Expose API port
EXPOSE 8080

# Default to running the API server
CMD ["./scraper-api", "-addr", ":8080", "-db", "/app/data/scraper.db"]
