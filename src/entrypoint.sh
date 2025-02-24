#!/bin/sh

# Function to format schedule message
format_schedule_message() {
    current_time=$(date +%s)
    target_time=$1
    remaining_seconds=$(( target_time - current_time ))
    
    echo "⏳ Next backup in $remaining_seconds seconds at $(date -d @$target_time '+%Y-%m-%d %H:%M:%S UTC')"
}

# Function to calculate next schedule time
calculate_next_time() {
    current_minute=$(date +%M)
    next_minute=$(( (current_minute + 1) % 60 ))
    next_hour=$(date +%H)
    if [ "$next_minute" -lt "$current_minute" ]; then
        next_hour=$(( (next_hour + 1) % 24 ))
    fi
    
    date -d "$(date +%Y-%m-%d) $next_hour:$next_minute:00" +%s
}

# Function to create cron job
setup_cron() {
    # Ensure crontabs directory exists
    mkdir -p /var/spool/cron/crontabs
    
    # Create a script that will be executed by cron
    cat > /usr/local/bin/backup-job.sh << EOF
#!/bin/sh

# Function to calculate next execution
calculate_next_time() {
    current_minute=\$(date +%M)
    next_minute=\$(( (current_minute + 1) % 60 ))
    next_hour=\$(date +%H)
    if [ "\$next_minute" -lt "\$current_minute" ]; then
        next_hour=\$(( (next_hour + 1) % 24 ))
    fi
    
    date -d "\$(date +%Y-%m-%d) \$next_hour:\$next_minute:00" +%s
}

# Function to format schedule message
format_schedule_message() {
    current_time=\$(date +%s)
    target_time=\$1
    remaining_seconds=\$(( target_time - current_time ))
    
    echo "⏳ Next backup in \$remaining_seconds seconds at \$(date -d @\$target_time '+%Y-%m-%d %H:%M:%S UTC')"
}

(
    echo "=== Backup started at \$(date '+%Y-%m-%d %H:%M:%S UTC') ==="
    /usr/local/bin/volback \\
        -containers='$(echo "${CONTAINERS}" | sed 's/\$/\\$/g')' \\
        -dropbox-refresh-token='${DROPBOX_REFRESH_TOKEN}' \\
        -dropbox-client-id='${DROPBOX_CLIENT_ID}' \\
        -dropbox-client-secret='${DROPBOX_CLIENT_SECRET}' \\
        -dropbox-path='${DROPBOX_PATH}' \\
        -keep-daily=${KEEP_DAILY} \\
        -keep-weekly=${KEEP_WEEKLY} \\
        -keep-monthly=${KEEP_MONTHLY} \\
        -keep-yearly=${KEEP_YEARLY}
    echo "=== Backup completed at \$(date '+%Y-%m-%d %H:%M:%S UTC') ==="
    next_time=\$(calculate_next_time)
    format_schedule_message "\$next_time"
    echo
) >> /var/log/volback.log 2>&1
EOF

    # Make the script executable
    chmod +x /usr/local/bin/backup-job.sh
    
    # Create cron entry
    echo "${CRON_SCHEDULE} /usr/local/bin/backup-job.sh" > /var/spool/cron/crontabs/root
    
    # Set proper permissions
    chmod 0644 /var/spool/cron/crontabs/root
    
    # Clear existing log file
    > /var/log/volback.log
    
    # Start log tailing first
    tail -F /var/log/volback.log &
    
    # Show initial schedule information
    echo "🕒 Starting scheduled backup service"
    echo "📝 Schedule: ${CRON_SCHEDULE}"
    echo "📋 Log file: /var/log/volback.log"
    next_time=$(calculate_next_time)
    format_schedule_message "$next_time"
    echo
    
    # Start crond and wait
    /usr/sbin/crond -f -L /dev/stdout
}

# Function for immediate execution
run_immediate() {
    echo "▶️ Starting immediate backup..."
    exec /usr/local/bin/volback \
        -containers="${CONTAINERS}" \
        -dropbox-refresh-token="${DROPBOX_REFRESH_TOKEN}" \
        -dropbox-client-id="${DROPBOX_CLIENT_ID}" \
        -dropbox-client-secret="${DROPBOX_CLIENT_SECRET}" \
        -dropbox-path="${DROPBOX_PATH}" \
        -keep-daily="${KEEP_DAILY}" \
        -keep-weekly="${KEEP_WEEKLY}" \
        -keep-monthly="${KEEP_MONTHLY}" \
        -keep-yearly="${KEEP_YEARLY}"
}

# Check if CRON_SCHEDULE is set
if [ -n "$CRON_SCHEDULE" ]; then
    setup_cron
else
    run_immediate
fi