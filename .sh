repomix --no-file-summary --no-security-check \
  --include "src/Dockerfile,src/backup.go,src/docker.go,src/dropbox.go,src/logger.go,src/main.go,src/retention.go,src/types.go" \
  --output "repopack.yml"

go mod tidy

docker build -t dublok/volback:latest -f src/Dockerfile .