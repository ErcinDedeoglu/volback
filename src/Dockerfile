# Build stage
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /volback .

# Final stage
FROM ubuntu:24.04

# Install required packages
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y \
        curl \
        wget \
        gpg \
        docker.io \
        busybox \
        cron \
        mariadb-client \
        unixodbc \
        apt-transport-https \
        gnupg2 \
        lsb-release \
        bash \
        procps \
        net-tools \
        iproute2 \
        locales \
        strace \
        lsof \
        vim \
        less \
        nano \
        htop \
        tzdata \
        ca-certificates \
        --no-install-recommends && \
    rm -rf /var/lib/apt/lists/*

# Add Microsoft repository and install new cross-platform sqlcmd
# WORKAROUND: Use the Ubuntu 22.04 (jammy) Microsoft repo for sqlcmd, since 24.04 repo may not provide the package yet.
RUN wget -qO- https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > /usr/share/keyrings/microsoft-prod.gpg && \
    echo "deb [arch=amd64,arm64,armhf signed-by=/usr/share/keyrings/microsoft-prod.gpg] https://packages.microsoft.com/ubuntu/22.04/prod jammy main" > /etc/apt/sources.list.d/microsoft-prod.list && \
    apt-get update && \
    apt-get install -y sqlcmd && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Add PostgreSQL official repository and install latest PostgreSQL client
RUN wget -qO- https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor > /usr/share/keyrings/postgresql.gpg && \
    echo "deb [arch=amd64,arm64 signed-by=/usr/share/keyrings/postgresql.gpg] https://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/postgresql.list && \
    apt-get update && \
    apt-get install -y postgresql-client && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Add mssql-tools to PATH for all users
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
