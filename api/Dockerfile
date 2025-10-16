# syntax=docker/dockerfile:1

# ----------- Build Stage -----------
ARG GO_VERSION=1.22.1
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum first for dependency caching
COPY --link go.mod go.sum ./

# Download dependencies (using cache mounts for speed)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source code
COPY --link . .

# Build the Go binary (static build, strip debug info)
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api-server ./main.go

# ----------- Final Stage -----------
FROM alpine:latest AS final

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the built binary from builder
COPY --from=builder /app/api-server ./api-server

# Copy migrations (if needed at runtime)
COPY --from=builder /app/internal/database/migrations ./internal/database/migrations

# Expose the default port (update if your app uses a different port)
EXPOSE 8080

USER appuser

# Entrypoint
CMD ["./api-server"]
