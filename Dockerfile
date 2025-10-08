# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application (compile cmd/webhook to feishu-github-tracker)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o feishu-github-tracker \
    ./cmd/feishu-github-tracker

# Final stage
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/feishu-github-tracker /app/feishu-github-tracker

# Copy configuration files (for production use without volume mounts)
COPY --from=builder /build/configs /app/configs

# Set working directory
WORKDIR /app

# Expose port (keep consistent with server.yaml / docker-compose)
EXPOSE 4594

# Run the application
ENTRYPOINT ["/app/feishu-github-tracker", "--reload"]
