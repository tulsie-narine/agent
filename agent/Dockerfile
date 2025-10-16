# syntax=docker/dockerfile:1

ARG GO_VERSION=1.22.1
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum first for dependency resolution
COPY --link go.mod go.sum ./

# Download dependencies (using cache mounts for speed)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source code
COPY --link . .

# Build the Go binary (static build, strip debug info)
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o agent ./main.go

# Final minimal image
FROM alpine:latest AS final

# Create non-root user
RUN addgroup -S agentgroup && adduser -S agentuser -G agentgroup

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy the built binary from builder
COPY --link --from=builder /app/agent ./agent

# Optionally copy example config (not secrets)
COPY --link config.example.json ./config.example.json

USER agentuser

# Set entrypoint
ENTRYPOINT ["/app/agent"]

# (Optional) Document the default port if your app listens on one
# EXPOSE 8080

# Healthcheck example (uncomment and adjust if needed)
# HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/app/agent", "health"]
