# Build stage
FROM golang:1.23-alpine AS builder
# Install required build dependencies
RUN apk add --no-cache gcc musl-dev
# Set working directory
WORKDIR /app
# Copy all files from current directory
COPY . .
# Download dependencies
RUN go mod download
# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /volback .

# Final stage
FROM alpine:3.19

# Environment variables
ENV CONTAINER=""
ENV BACKUP_ID=""
ENV KEEP_DAILY="0"
ENV KEEP_WEEKLY="0" 
ENV KEEP_MONTHLY="0"
ENV KEEP_YEARLY="0"
ENV DROPBOX_REFRESH_TOKEN=""
ENV DROPBOX_CLIENT_ID=""
ENV DROPBOX_CLIENT_SECRET=""
ENV DROPBOX_PATH=""
ENV CRON_SCHEDULE=""

# Install required packages
RUN apk add --no-cache \
    docker-cli \
    tzdata \
    dcron

# Copy the binary from builder
COPY --from=builder /volback /usr/local/bin/

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Create directory for temporary files
RUN mkdir -p /tmp/backups

# Set working directory
WORKDIR /backups

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]