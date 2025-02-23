#!/bin/sh

# Function to run backup
run_backup() {
    volback \
        --container "${CONTAINER}" \
        --id "${BACKUP_ID}" \
        --stop "${STOP_CONTAINER}" \
        --keep-daily "${KEEP_DAILY}" \
        --keep-weekly "${KEEP_WEEKLY}" \
        --keep-monthly "${KEEP_MONTHLY}" \
        --keep-yearly "${KEEP_YEARLY}" \
        --dropbox-refresh-token "${DROPBOX_REFRESH_TOKEN}" \
        --dropbox-client-id "${DROPBOX_CLIENT_ID}" \
        --dropbox-client-secret "${DROPBOX_CLIENT_SECRET}" \
        --dropbox-path "${DROPBOX_PATH}"
}

# If CRON_SCHEDULE is set, configure cron
if [ -n "${CRON_SCHEDULE}" ]; then
    # Create cron job
    echo "${CRON_SCHEDULE} /entrypoint.sh run" > /etc/cron.d/backup-cron
    chmod 0644 /etc/cron.d/backup-cron
    crontab /etc/cron.d/backup-cron

    # Run initial backup
    run_backup

    # Start cron daemon
    echo "🔄 Starting cron scheduler with schedule: ${CRON_SCHEDULE}"
    crond -f -l 2
else
    # Just run backup once
    run_backup
fi

# Handle the "run" command for cron execution
if [ "$1" = "run" ]; then
    run_backup
fi