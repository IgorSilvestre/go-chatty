# Multi-stage build for go-chatty API
# Builder stage
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache ca-certificates tzdata git

# Pre-cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a statically linked binary
ENV CGO_ENABLED=0
RUN go build -o /out/api ./cmd/api

# Runtime stage
FROM alpine:3.20

# Add CA certs and timezone data
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -H -u 10001 appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /out/api /usr/local/bin/api

# Use an unprivileged user
USER appuser

EXPOSE 8080
ENV GIN_MODE=release

# Start the API
CMD ["/usr/local/bin/api"]
