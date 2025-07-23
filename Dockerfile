# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for private repos and SSL)
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

# Create appuser for security
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bookwork-api \
    cmd/api/main.go

# Production stage
FROM scratch

# Import CA certificates and user from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /build/bookwork-api /bookwork-api

# Use appuser
USER appuser

# Expose port
EXPOSE 8000

# Set the entrypoint
ENTRYPOINT ["/bookwork-api"]
