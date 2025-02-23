repomix --no-file-summary --no-security-check \
  --include "src/Dockerfile,src/backup.go,src/docker.go,src/dropbox.go,src/logger.go,src/main.go,src/retention.go,src/types.go" \
  --output "repopack.yml"

go mod tidy

docker build -t dublok/volback:latest -f src/Dockerfile ./src

go run ./*.go --container test1 \
    --id a-unique-name-to-identify-backup \
    --keep-daily 1 --keep-weekly 1 --keep-monthly 1 --keep-yearly 1 \
    --dropbox-refresh-token "token" \
    --dropbox-client-id "id" \
    --dropbox-client-secret "secret" \
    --dropbox-path "/backups/docker"

docker run --rm -it \
  -e CONTAINER=test1 \
  -e BACKUP_ID=a-unique-name \
  -e KEEP_DAILY=1 \
  -e KEEP_WEEKLY=1 \
  -e KEEP_MONTHLY=1 \
  -e KEEP_YEARLY=1 \
  -e DROPBOX_REFRESH_TOKEN=xxx \
  -e DROPBOX_CLIENT_ID=xxx \
  -e DROPBOX_CLIENT_SECRET=xxx \
  -e DROPBOX_PATH=/backups/docker \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp:/tmp \
  dublok/volback:latest


docker run -d \
  -e CONTAINER=test1 \
  -e BACKUP_ID=a-unique-name \
  -e KEEP_DAILY=1 \
  -e KEEP_WEEKLY=1 \
  -e KEEP_MONTHLY=1 \
  -e KEEP_YEARLY=1 \
  -e DROPBOX_REFRESH_TOKEN=xxx \
  -e DROPBOX_CLIENT_ID=xxx \
  -e DROPBOX_CLIENT_SECRET=xxx \
  -e DROPBOX_PATH=/backups/docker \
  -e CRON_SCHEDULE="0 0 * * *" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp:/tmp \
  dublok/volback:latest