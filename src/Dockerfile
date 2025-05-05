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
        docker.io \
        busybox \
        cron \
        mariadb-client \
        unixodbc \
        apt-transport-https \
        gnupg2 \
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

# Add Microsoft repository and install MSSQL tools
RUN curl https://packages.microsoft.com/config/ubuntu/24.04/packages-microsoft-prod.deb -o /tmp/msprod.deb && \
    echo "Downloaded Microsoft repository package" && \
    dpkg -i /tmp/msprod.deb && \
    echo "Installed Microsoft repository" && \
    rm /tmp/msprod.deb && \
    apt-get update && \
    echo "Updated package lists with Microsoft repository" && \
    ACCEPT_EULA=Y apt-get install -y mssql-tools18 && \
    echo "Installed SQL Server tools" && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Add mssql-tools to PATH for all users
ENV PATH="${PATH}:/opt/mssql-tools18/bin"
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
