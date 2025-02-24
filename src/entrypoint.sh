#!/bin/sh

# Source shared functions
. /usr/local/bin/functions.sh

# Function to create cron job
setup_cron() {
    # Ensure crontabs directory exists
    mkdir -p /var/spool/cron/crontabs
    
    # Create a script that will be executed by cron
    cat > /usr/local/bin/backup-job.sh << EOF
#!/bin/sh
. /usr/local/bin/functions.sh

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