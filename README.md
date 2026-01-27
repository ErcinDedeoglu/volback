# Volback

Automated backup utility for Docker volumes, host directories, and databases with Dropbox storage and intelligent retention policies.

[![Docker Image](https://img.shields.io/docker/v/dublok/volback?sort=semver&label=Docker%20Hub)](https://hub.docker.com/r/dublok/volback)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

## What It Does

Volback backs up your Docker container volumes, host directories, and databases to Dropbox automatically. It handles the entire backup lifecycle: creating compressed archives, uploading to cloud storage, and cleaning up old backups based on your retention policy.

**Supported Backup Types:**
- Host directories (any path on the system)
- Docker container volumes (compressed with 7z)
- MySQL / MariaDB databases
- PostgreSQL databases  
- Microsoft SQL Server databases
- Qdrant vector database collections

## Quick Start

### One-Time Backup

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp:/tmp \
  -e CONTAINERS='[{"container":"my-app"}]' \
  -e DROPBOX_REFRESH_TOKEN="your-refresh-token" \
  -e DROPBOX_CLIENT_ID="your-client-id" \
  -e DROPBOX_CLIENT_SECRET="your-client-secret" \
  -e DROPBOX_PATH="/backups" \
  dublok/volback:latest
```

### Scheduled Backup (Daily at Midnight)

```bash
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp:/tmp \
  -e CONTAINERS='[{"container":"my-app"}]' \
  -e DROPBOX_REFRESH_TOKEN="your-refresh-token" \
  -e DROPBOX_CLIENT_ID="your-client-id" \
  -e DROPBOX_CLIENT_SECRET="your-client-secret" \
  -e DROPBOX_PATH="/backups" \
  -e CRON_SCHEDULE="0 0 * * *" \
  -e KEEP_DAILY=7 \
  -e KEEP_WEEKLY=4 \
  -e KEEP_MONTHLY=6 \
  dublok/volback:latest
```

## Dropbox Setup

Before using Volback, you need to create a Dropbox app and obtain OAuth credentials.

### Step 1: Create a Dropbox App

1. Go to [Dropbox App Console](https://www.dropbox.com/developers/apps)
2. Click **Create app**
3. Choose **Scoped access**
4. Choose **Full Dropbox** or **App folder** (depending on your preference)
5. Name your app (e.g., "volback")
6. Click **Create app**

### Step 2: Configure Permissions

1. Go to the **Permissions** tab
2. Enable these permissions:
   - `files.content.write`
   - `files.content.read`
3. Click **Submit**

### Step 3: Generate Refresh Token

1. Go to the **Settings** tab
2. Note your **App key** (this is your `DROPBOX_CLIENT_ID`)
3. Note your **App secret** (this is your `DROPBOX_CLIENT_SECRET`)
4. Under **OAuth 2**, set **Access token expiration** to **No expiration** if available, or use refresh tokens
5. Generate a refresh token using the OAuth flow or [Dropbox OAuth Guide](https://developers.dropbox.com/oauth-guide)

## Configuration

All configuration is done via environment variables.

### Required Variables

| Variable | Description |
|----------|-------------|
| `DROPBOX_REFRESH_TOKEN` | OAuth refresh token from your Dropbox app |
| `DROPBOX_CLIENT_ID` | App key from your Dropbox app |
| `DROPBOX_CLIENT_SECRET` | App secret from your Dropbox app |
| `DROPBOX_PATH` | Destination folder in Dropbox (e.g., `/backups`) |

### Backup Targets

At least one backup target must be configured. All targets use JSON array format.

#### Docker Containers

```bash
CONTAINERS='[
  {"container": "my-app"},
  {"container": "nginx", "stop": true},
  {"container": "postgres", "backup_id": "db-server", "depends_on": ["my-app"]}
]'
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `container` | string | Yes | - | Container name |
| `stop` | boolean | No | `false` | Stop container during backup |
| `backup_id` | string | No | container name | Custom identifier for backup folder |
| `include_ro` | boolean | No | `false` | Include read-only volumes |
| `depends_on` | array | No | `[]` | Process these containers first |

#### MySQL / MariaDB

```bash
MYSQL='[
  {
    "container": "mysql",
    "user": "root",
    "password": "secret",
    "databases": ["app_db", "analytics"]
  }
]'
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `container` | string | Yes | - | MySQL container name |
| `user` | string | Yes | - | Database user |
| `password` | string | Yes | - | Database password |
| `databases` | array | Yes | - | List of databases to backup |
| `port` | integer | No | `3306` | MySQL port |
| `backup_id` | string | No | container name | Custom backup folder name |

#### PostgreSQL

```bash
POSTGRESQL='[
  {
    "container": "postgres",
    "user": "postgres",
    "password": "secret",
    "databases": ["mydb"]
  }
]'
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `container` | string | Yes | - | PostgreSQL container name |
| `user` | string | Yes | - | Database user |
| `password` | string | Yes | - | Database password |
| `databases` | array | Yes | - | List of databases to backup |
| `port` | integer | No | `5432` | PostgreSQL port |
| `backup_id` | string | No | container name | Custom backup folder name |

#### Microsoft SQL Server

```bash
MSSQL='[
  {
    "host": "mssql-server",
    "user": "sa",
    "password": "YourStrong!Passw0rd",
    "databases": ["master", "app_db"]
  }
]'
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | Yes | - | SQL Server hostname |
| `user` | string | Yes | - | Database user |
| `password` | string | Yes | - | Database password |
| `databases` | array | Yes | - | List of databases to backup |
| `port` | integer | No | `1433` | SQL Server port |
| `backup_id` | string | No | host name | Custom backup folder name |

#### Qdrant Vector Database

```bash
QDRANT='[
  {
    "host": "qdrant",
    "port": 6333,
    "collections": ["embeddings", "documents"]
  }
]'
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | Yes | - | Qdrant hostname |
| `collections` | array | Yes | - | List of collections to backup |
| `port` | integer | No | `6333` | Qdrant HTTP port |
| `api_key` | string | No | - | API key if authentication is enabled |
| `backup_id` | string | No | host name | Custom backup folder name |

#### Host Paths

Backup any directory on the host system directly, without needing a Docker container.

```bash
PATHS='[
  {"path": "/home/user/data"},
  {"path": "/etc/myapp", "backup_id": "myapp-config"},
  {"path": "/var/lib/important"}
]'
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | - | Absolute path to backup |
| `backup_id` | string | No | directory name | Custom backup folder name |

**Note:** When using `PATHS`, mount the directories into the container:

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp:/tmp \
  -v /home/user/data:/home/user/data:ro \
  -e PATHS='[{"path": "/home/user/data", "backup_id": "my-data"}]' \
  -e DROPBOX_REFRESH_TOKEN="..." \
  -e DROPBOX_CLIENT_ID="..." \
  -e DROPBOX_CLIENT_SECRET="..." \
  -e DROPBOX_PATH="/backups" \
  dublok/volback:latest
```

### Retention Policy

Control how many backups to keep. Volback automatically deletes old backups that exceed these limits.

| Variable | Description |
|----------|-------------|
| `KEEP_DAILY` | Number of daily backups to retain |
| `KEEP_WEEKLY` | Number of weekly backups to retain |
| `KEEP_MONTHLY` | Number of monthly backups to retain |
| `KEEP_YEARLY` | Number of yearly backups to retain |

**How it works:**
- The most recent backup is always kept
- One backup per day/week/month/year is retained (the oldest in each period)
- Backups outside these windows are deleted

**Example:** With `KEEP_DAILY=7, KEEP_WEEKLY=4, KEEP_MONTHLY=6`:
- Keep backups from the last 7 different days
- Keep 4 weekly backups (one per week)
- Keep 6 monthly backups (one per month)

### Scheduling

| Variable | Description |
|----------|-------------|
| `CRON_SCHEDULE` | Cron expression for scheduled backups |

If `CRON_SCHEDULE` is set, Volback runs as a daemon and executes backups on schedule. If not set, it runs once and exits.

**Common schedules:**
- `0 0 * * *` - Daily at midnight
- `0 */6 * * *` - Every 6 hours
- `0 2 * * 0` - Weekly on Sunday at 2 AM
- `30 1 1 * *` - Monthly on the 1st at 1:30 AM

## Docker Compose Example

```yaml
services:
  volback:
    image: dublok/volback:latest
    container_name: volback
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /tmp:/tmp
    environment:
      # Dropbox credentials
      DROPBOX_REFRESH_TOKEN: ${DROPBOX_REFRESH_TOKEN}
      DROPBOX_CLIENT_ID: ${DROPBOX_CLIENT_ID}
      DROPBOX_CLIENT_SECRET: ${DROPBOX_CLIENT_SECRET}
      DROPBOX_PATH: /backups/myserver
      
      # Schedule (daily at 3 AM)
      CRON_SCHEDULE: "0 3 * * *"
      
      # Retention
      KEEP_DAILY: 7
      KEEP_WEEKLY: 4
      KEEP_MONTHLY: 12
      KEEP_YEARLY: 2
      
      # Backup targets
      CONTAINERS: |
        [
          {"container": "app", "stop": true},
          {"container": "redis"}
        ]
      MYSQL: |
        [
          {
            "container": "mysql",
            "user": "root",
            "password": "${MYSQL_ROOT_PASSWORD}",
            "databases": ["app_production"]
          }
        ]
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Docker Host    │     │    Volback      │     │    Dropbox      │
│                 │     │                 │     │                 │
│  ┌───────────┐  │     │  1. Stop        │     │  ┌───────────┐  │
│  │ Container │◄─┼─────┼─────────────────┼─────┼─►│ /backups  │  │
│  └───────────┘  │     │  2. Backup      │     │  │  /app     │  │
│  ┌───────────┐  │     │  3. Compress    │     │  │  /mysql   │  │
│  │  Volumes  │◄─┼─────┼─────────────────┼─────┼─►│  /...     │  │
│  └───────────┘  │     │  4. Upload      │     │  └───────────┘  │
│  ┌───────────┐  │     │  5. Start       │     │                 │
│  │ Database  │◄─┼─────┼─────────────────┼─────┼─►  Retention    │
│  └───────────┘  │     │  6. Cleanup     │     │   Management    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Backup File Structure

Backups are organized in Dropbox by backup ID and timestamped:

```
/backups/
├── my-app/
│   ├── 20260127.030000.7z
│   ├── 20260126.030000.7z
│   └── 20260125.030000.7z
├── my-data/
│   ├── 20260127.030000.7z
│   └── 20260126.030000.7z
├── mysql/
│   ├── app_db/
│   │   ├── 20260127.030000.sql
│   │   └── 20260126.030000.sql
│   └── analytics/
│       └── 20260127.030000.sql
└── postgres/
    └── mydb/
        └── 20260127.030000.sql
```

## Troubleshooting

### Backup fails with "Dropbox configuration is required"
Ensure all three Dropbox variables are set: `DROPBOX_REFRESH_TOKEN`, `DROPBOX_CLIENT_ID`, and `DROPBOX_CLIENT_SECRET`.

### Container not found
Make sure the container name matches exactly (case-sensitive) and the container exists on the Docker host.

### Permission denied errors
The Docker socket must be mounted: `-v /var/run/docker.sock:/var/run/docker.sock`

### Large files fail to upload
Files over 150MB use chunked uploads automatically. Ensure stable network connectivity.

### Scheduled backup not running
Check container logs: `docker logs volback`. Verify the cron expression is valid.

## Requirements

- Docker 19.03+
- Dropbox account with API access
- Network access to Dropbox API endpoints

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
