# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /volback .

# Final stage
FROM alpine:3.19

# Install required packages
RUN apk add --no-cache docker-cli busybox-suid

# Copy binary from builder
COPY --from=builder /volback /usr/local/bin/

# Copy shell scripts
COPY entrypoint.sh /entrypoint.sh
COPY functions.sh /usr/local/bin/functions.sh

# Set execute permissions
RUN chmod +x /entrypoint.sh && \
    chmod +x /usr/local/bin/functions.sh

# Create directories and set permissions
RUN mkdir -p /var/spool/cron/crontabs && \
    mkdir -p /var/log && \
    touch /var/log/volback.log && \
    chmod 777 /var/log/volback.log && \
    chmod 600 /var/spool/cron/crontabs

# Set working directory
WORKDIR /backups

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]