#!/bin/sh

# Function to format schedule message
format_schedule_message() {
    current_time=$(date +%s)
    target_time=$1
    remaining_seconds=$(( target_time - current_time ))
    
    echo "🕐 Current time: $(date '+%Y-%m-%d %H:%M:%S UTC')"
    echo "⏳ Next backup in $remaining_seconds seconds at $(date -d @$target_time '+%Y-%m-%d %H:%M:%S UTC')"
}

# Function to calculate next schedule time
calculate_next_time() {
    # Parse the cron schedule from environment variable
    schedule="${CRON_SCHEDULE}"
    
    # Use busybox date to get next scheduled time
    current_timestamp=$(date +%s)
    next_timestamp=$(busybox date -d "$(busybox date -d "@$current_timestamp" "+\
        $(echo "$schedule" | awk '{printf "+%s minute +%s hour +%s day +%s month +%s weekday", 
        $1=="*"?0:$1, 
        $2=="*"?0:$2,
        $3=="*"?0:$3,
        $4=="*"?0:$4,
        $5=="*"?0:$5}')")" +%s)

    # If the calculated time is in the past, add a day
    if [ "$next_timestamp" -le "$current_timestamp" ]; then
        next_timestamp=$(busybox date -d "$(busybox date -d "@$next_timestamp" "+%Y-%m-%d") + 1 day" +%s)
    fi

    echo "$next_timestamp"
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
    # Parse the cron schedule from environment variable
    schedule="${CRON_SCHEDULE}"
    
    # Use busybox date to get next scheduled time
    current_timestamp=\$(date +%s)
    next_timestamp=\$(busybox date -d "\$(busybox date -d "@\$current_timestamp" "+\
        \$(echo "\$schedule" | awk '{printf "+%s minute +%s hour +%s day +%s month +%s weekday", 
        \$1=="*"?0:\$1, 
        \$2=="*"?0:\$2,
        \$3=="*"?0:\$3,
        \$4=="*"?0:\$4,
        \$5=="*"?0:\$5}')")" +%s)

    # If the calculated time is in the past, add a day
    if [ "\$next_timestamp" -le "\$current_timestamp" ]; then
        next_timestamp=\$(busybox date -d "\$(busybox date -d "@\$next_timestamp" "+%Y-%m-%d") + 1 day" +%s)
    fi

    echo "\$next_timestamp"
}

# Function to format schedule message
format_schedule_message() {
    current_time=\$(date +%s)
    target_time=\$1
    remaining_seconds=\$(( target_time - current_time ))
    
    echo "🕐 Current time: \$(date '+%Y-%m-%d %H:%M:%S UTC')"
    echo "⏳ Next backup in \$remaining_seconds seconds at \$(date -d @\$target_time '+%Y-%m-%d %H:%M:%S UTC')"
}

(
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔄 Backup Process Started"
    echo "🕐 Current time: \$(date '+%Y-%m-%d %H:%M:%S UTC')"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    /usr/local/bin/volback \\
        -containers='${CONTAINERS}' \\
        -dropbox-refresh-token='${DROPBOX_REFRESH_TOKEN}' \\
        -dropbox-client-id='${DROPBOX_CLIENT_ID}' \\
        -dropbox-client-secret='${DROPBOX_CLIENT_SECRET}' \\
        -dropbox-path='${DROPBOX_PATH}' \\
        -keep-daily=${KEEP_DAILY} \\
        -keep-weekly=${KEEP_WEEKLY} \\
        -keep-monthly=${KEEP_MONTHLY} \\
        -keep-yearly=${KEEP_YEARLY}
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ Backup Process Completed"
    next_time=\$(calculate_next_time)
    format_schedule_message "\$next_time"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
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
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🕒 Backup Service Started"
    echo "📝 Schedule: ${CRON_SCHEDULE}"
    echo "📋 Log file: /var/log/volback.log"
    next_time=$(calculate_next_time)
    format_schedule_message "$next_time"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
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