# VOLBACK KNOWLEDGE BASE

**Generated:** 2026-01-27  
**Commit:** 5cdd968  
**Branch:** v1.0

## OVERVIEW

Docker volume backup utility (Go) supporting container volumes, MySQL, PostgreSQL, MSSQL, and Qdrant databases with Dropbox storage and retention policies.

## STRUCTURE

```
volback/
├── src/                    # All Go code + Docker artifacts
│   ├── main.go             # Entry point, flag parsing, orchestration
│   ├── types.go            # All config/result struct definitions
│   ├── backup_manager.go   # Container backup with dependency resolution
│   ├── backup.go           # Volume backup logic (tar, 7z)
│   ├── *_backup.go         # Database-specific backup handlers
│   ├── dropbox.go          # Dropbox API client (chunked upload, retention)
│   ├── docker.go           # Docker SDK operations
│   ├── retention.go        # Daily/weekly/monthly/yearly retention
│   ├── logger.go           # Emoji-prefixed logging (logHeader/logStep/logSubStep)
│   ├── Dockerfile          # Multi-stage: golang:1.23 -> ubuntu:24.04
│   ├── entrypoint.sh       # Cron vs immediate execution mode
│   └── functions.sh        # Shell helpers for scheduling display
├── .github/workflows/      # CI: build-and-push on src/** changes
└── .sh                     # Dev script: build, run, push examples
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add new backup type | `src/{type}_backup.go` | Follow mysql_backup.go pattern |
| Modify retention logic | `src/retention.go` | Daily/weekly/monthly/yearly |
| Change Dropbox upload | `src/dropbox.go` | Chunked upload for >150MB |
| Add new config fields | `src/types.go` | JSON tags: snake_case, `omitempty` for optional |
| Container lifecycle | `src/docker.go` | Uses Docker SDK, not CLI |
| Cron scheduling | `src/entrypoint.sh` | Creates backup-job.sh dynamically |
| CI/CD | `.github/workflows/build-and-push.yml` | Uses reusable workflow from dubloksoftware/workflows |

## CONVENTIONS

### Naming
- **Files**: `snake_case.go` (e.g., `mysql_backup.go`)
- **Types**: PascalCase, slice types pluralized (`MySQLConfig` -> `MySQLConfigs`)
- **Functions**: `processXBackups()`, `dumpDatabase()`, `getBackupID()`
- **JSON tags**: snake_case (`backup_id`), optional fields use pointer + `omitempty`

### Logging
```go
logHeader("=== Section ===")     // Major sections
logStep("📦 Processing: %s", x)  // Primary operations
logSubStep("Details: %s", y)     // Indented details
```

### Error Handling
```go
if err != nil {
    logStep("❌ Failed to X: %v", err)
    return fmt.Errorf("failed to X for %s: %w", id, err)
}
```

### Error Aggregation (continue on partial failures)
```go
var errorList []string
for _, item := range items {
    if err := process(item); err != nil {
        errorList = append(errorList, fmt.Sprintf("%s: %v", item.Name, err))
        continue  // Don't fail entire batch
    }
}
if len(errorList) > 0 {
    return fmt.Errorf("errors: %s", strings.Join(errorList, "; "))
}
```

## ANTI-PATTERNS

| Don't | Why | Instead |
|-------|-----|---------|
| Use Docker CLI in Go code | SDK provides better error handling | Use `github.com/docker/docker/client` |
| Hardcode backup extensions | Retention checks .7z, .sql, .bak | Add to `dropbox.go:ListFiles()` filter |
| Skip `omitempty` on optional JSON fields | Will serialize nil as null | Use `*Type` + `json:"field,omitempty"` |
| os.Exit() in library functions | Prevents error aggregation | Return error, let main() handle exit |
| Delete backups synchronously | Slow for many files | Consider batch deletion (not implemented) |

## COMMANDS

```bash
# Development
cd src && go mod tidy
go run ./*.go --containers '[...]' --dropbox-refresh-token xxx --dropbox-client-id xxx --dropbox-client-secret xxx --dropbox-path /backups

# Docker build
docker build -t dublok/volback:latest -f src/Dockerfile ./src
docker push dublok/volback:latest

# Run immediate backup
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v /tmp:/tmp \
  -e CONTAINERS='[{"container":"myapp"}]' \
  -e DROPBOX_REFRESH_TOKEN=xxx -e DROPBOX_CLIENT_ID=xxx -e DROPBOX_CLIENT_SECRET=xxx \
  -e DROPBOX_PATH=/backups -e KEEP_DAILY=7 \
  dublok/volback:latest

# Run scheduled backup
docker run -d ... -e CRON_SCHEDULE="0 0 * * *" dublok/volback:latest
```

## RUNTIME CONFIG

### Required
- `DROPBOX_REFRESH_TOKEN`, `DROPBOX_CLIENT_ID`, `DROPBOX_CLIENT_SECRET` - OAuth credentials
- `DROPBOX_PATH` - Destination folder (e.g., `/backups`)

### Backup Targets (JSON arrays, at least one required)
- `CONTAINERS` - Docker volume backups: `[{"container":"name","stop":true,"backup_id":"custom-id"}]`
- `MYSQL` - `[{"container":"mysql","user":"root","password":"x","databases":["db1"]}]`
- `POSTGRESQL` - Same pattern, port defaults to 5432
- `MSSQL` - Uses `host` instead of `container`, port defaults to 1433
- `QDRANT` - `[{"host":"localhost","port":6333,"collections":["col1"]}]`

### Retention
- `KEEP_DAILY`, `KEEP_WEEKLY`, `KEEP_MONTHLY`, `KEEP_YEARLY` - Integer counts

### Scheduling
- `CRON_SCHEDULE` - If set, runs as daemon; if empty, runs once and exits

## NOTES

- **Dropbox mandatory**: No fallback storage; app exits if credentials missing
- **Lock file**: `/var/run/volback.lock` prevents concurrent backups
- **Temp files**: Uses `/tmp/volback-{container}-{timestamp}/`, cleaned up after upload
- **Large files**: >150MB uses chunked Dropbox upload session
- **Multi-platform**: CI builds linux/amd64 + linux/arm64
- **No tests**: Zero `*_test.go` files exist; test manually with Docker
- **Ubuntu 24.04 quirk**: Uses Ubuntu 22.04 Microsoft repo for sqlcmd (24.04 repo lacks it)
