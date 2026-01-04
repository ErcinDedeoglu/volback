#!/bin/sh

# Source shared functions
. /usr/local/bin/functions.sh

# Function to create cron job
setup_cron() {
    # Ensure crontabs directory exists
    mkdir -p /var/spool/cron/crontabs
    
    # Create a script that will be executed by cron
    cat > /usr/local/bin/backup-job.sh << 'EOFSCRIPT'
#!/bin/bash
. /usr/local/bin/functions.sh

# Output to both log file and stdout (for docker logs)
{
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔄 Backup Process Started"
    echo "🕐 Current time: $(date '+%Y-%m-%d %H:%M:%S UTC')"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    /usr/local/bin/volback \
        -containers="${CONTAINERS}" \
        -mysql="${MYSQL}" \
        -mssql="${MSSQL}" \
        -postgresql="${POSTGRESQL}" \
        -qdrant="${QDRANT}" \
        -dropbox-refresh-token="${DROPBOX_REFRESH_TOKEN}" \
        -dropbox-client-id="${DROPBOX_CLIENT_ID}" \
        -dropbox-client-secret="${DROPBOX_CLIENT_SECRET}" \
        -dropbox-path="${DROPBOX_PATH}" \
        -keep-daily="${KEEP_DAILY}" \
        -keep-weekly="${KEEP_WEEKLY}" \
        -keep-monthly="${KEEP_MONTHLY}" \
        -keep-yearly="${KEEP_YEARLY}"
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ Backup Process Completed"
    next_time=$(calculate_next_time)
    format_schedule_message "$next_time"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
} 2>&1 | tee -a /var/log/volback.log
EOFSCRIPT

    # Write environment variables to a file that the cron job can source
    cat > /usr/local/bin/backup-env.sh << EOF
export CONTAINERS='${CONTAINERS}'
export MYSQL='${MYSQL}'
export MSSQL='${MSSQL}'
export POSTGRESQL='${POSTGRESQL}'
export QDRANT='${QDRANT}'
export DROPBOX_REFRESH_TOKEN='${DROPBOX_REFRESH_TOKEN}'
export DROPBOX_CLIENT_ID='${DROPBOX_CLIENT_ID}'
export DROPBOX_CLIENT_SECRET='${DROPBOX_CLIENT_SECRET}'
export DROPBOX_PATH='${DROPBOX_PATH}'
export KEEP_DAILY='${KEEP_DAILY}'
export KEEP_WEEKLY='${KEEP_WEEKLY}'
export KEEP_MONTHLY='${KEEP_MONTHLY}'
export KEEP_YEARLY='${KEEP_YEARLY}'
export CRON_SCHEDULE='${CRON_SCHEDULE}'
EOF

    # Make the scripts executable
    chmod +x /usr/local/bin/backup-job.sh
    chmod +x /usr/local/bin/backup-env.sh
    
    # Create cron entry that sources the environment file first
    echo "${CRON_SCHEDULE} . /usr/local/bin/backup-env.sh && /usr/local/bin/backup-job.sh" > /var/spool/cron/crontabs/root
    
    # Set proper permissions
    chmod 0644 /var/spool/cron/crontabs/root
    
    # Clear existing log file
    > /var/log/volback.log
    
    # Show initial schedule information
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🕒 Backup Service Started"
    echo "📝 Schedule: ${CRON_SCHEDULE}"
    echo "📋 Log file: /var/log/volback.log"
    next_time=$(calculate_next_time)
    format_schedule_message "$next_time"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    
    # Start cron daemon in foreground (keeps container alive)
    exec /usr/sbin/cron -f -L 15
}

# Function for immediate execution
run_immediate() {
    echo "▶️ Starting immediate backup..."
    exec /usr/local/bin/volback \
        -containers="${CONTAINERS}" \
        -mysql="${MYSQL}" \
        -mssql="${MSSQL}" \
        -postgresql="${POSTGRESQL}" \
        -qdrant="${QDRANT}" \
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