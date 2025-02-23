# Build stage
FROM golang:1.23-alpine AS builder

# Install required build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy the entire src directory (including go.mod and go.sum)
COPY src/ ./

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /volback .

# Final stage
FROM alpine:3.19

# Install Docker client
RUN apk add --no-cache docker-cli

# Copy the binary from builder
COPY --from=builder /volback /usr/local/bin/

# Create directory for temporary files
RUN mkdir -p /tmp/backups

# Set working directory
WORKDIR /backups

# Set entrypoint
ENTRYPOINT ["volback"]