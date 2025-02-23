#!/bin/sh

# Execute the backup command with all arguments directly from environment variables
exec /usr/local/bin/volback \
  -containers="$CONTAINERS" \
  -dropbox-refresh-token="$DROPBOX_REFRESH_TOKEN" \
  -dropbox-client-id="$DROPBOX_CLIENT_ID" \
  -dropbox-client-secret="$DROPBOX_CLIENT_SECRET" \
  -dropbox-path="$DROPBOX_PATH" \
  -keep-daily="$KEEP_DAILY" \
  -keep-weekly="$KEEP_WEEKLY" \
  -keep-monthly="$KEEP_MONTHLY" \
  -keep-yearly="$KEEP_YEARLY"