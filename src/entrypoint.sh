#!/bin/sh

# Function to format next schedule time
format_next_schedule() {
    current_minute=$(date +%M)
    next_minute=$(( (current_minute + 1) % 60 ))
    next_hour=$(date +%H)
    if [ "$next_minute" -lt "$current_minute" ]; then
        next_hour=$(( (next_hour + 1) % 24 ))
    fi
    
    current_time=$(date +%s)
    next_time=$(date -d "$(date +%Y-%m-%d) $next_hour:$next_minute:00" +%s)
    remaining_seconds=$(( next_time - current_time ))
    
    echo "🕒 Starting scheduled backup service"
    echo "📝 Schedule: ${CRON_SCHEDULE}"
    echo "📋 Log file: /var/log/volback.log"
    echo "⏳ Next backup in $remaining_seconds seconds at $(date -d "$(date +%Y-%m-%d) $next_hour:$next_minute:00")"
    echo
}

# Function to create cron job
setup_cron() {
    # Ensure crontabs directory exists
    mkdir -p /var/spool/cron/crontabs
    
    # Create a script that will be executed by cron
    cat > /usr/local/bin/backup-job.sh << EOF
#!/bin/sh

# Function to calculate and display next execution
next_execution() {
    current_time=\$(date +%s)
    
    # Parse the cron schedule
    minute=\$(echo "$CRON_SCHEDULE" | awk '{print \$1}')
    hour=\$(echo "$CRON_SCHEDULE" | awk '{print \$2}')
    
    # Calculate next execution time
    if [ "\$minute" = "*" ]; then
        # If running every minute
        next_minute=\$(( (\$(date +%M) + 1) % 60 ))
        next_hour=\$(date +%H)
        if [ "\$next_minute" -eq 0 ]; then
            next_hour=\$(( (next_hour + 1) % 24 ))
        fi
        next_time=\$(date -d "\$(date +%Y-%m-%d) \$next_hour:\$next_minute:00" +%s)
    else
        # For specific minute
        next_time=\$(date -d "\$(date +%Y-%m-%d) \$(date +%H):\$minute:00" +%s)
        if [ \$next_time -le \$current_time ]; then
            next_time=\$(date -d "\$(date +%Y-%m-%d) \$((\$(date +%H)+1)):\$minute:00" +%s)
        fi
    fi
    
    # Calculate remaining time
    remaining_seconds=\$(( next_time - current_time ))
    
    echo "⏳ Next backup in \$remaining_seconds seconds at \$(date -d @\$next_time)"
}

(
    echo "=== Backup started at \$(date) ==="
    /usr/local/bin/volback \
        -containers='$CONTAINERS' \
        -dropbox-refresh-token='$DROPBOX_REFRESH_TOKEN' \
        -dropbox-client-id='$DROPBOX_CLIENT_ID' \
        -dropbox-client-secret='$DROPBOX_CLIENT_SECRET' \
        -dropbox-path='$DROPBOX_PATH' \
        -keep-daily=$KEEP_DAILY \
        -keep-weekly=$KEEP_WEEKLY \
        -keep-monthly=$KEEP_MONTHLY \
        -keep-yearly=$KEEP_YEARLY
    echo "=== Backup completed at \$(date) ==="
    next_execution
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
    format_next_schedule
    
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