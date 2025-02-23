# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /volback .

# Final stage
FROM alpine:3.19
RUN apk add --no-cache docker-cli
COPY --from=builder /volback /usr/local/bin/
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
WORKDIR /backups
ENTRYPOINT ["/entrypoint.sh"]