# 75 Hard Discord Bot

A vibe-coded Discord bot for managing the 75 Half Chub for Dads challenge with automatic progress tracking.
- Utilized [cursor-agent](https://cursor.com/docs/cli/overview)

## Usage

### Local Development
```bash
export DISCORD_BOT_TOKEN="your-bot-token"
export DISCORD_CHANNEL_ID="your-channel-id"
# Optional: export DB_HOST="localhost" DB_PASSWORD="your-password"
go run main.go
```

### Docker
```bash
docker build -t 75-hard-bot .
docker run -d \
  -e DISCORD_BOT_TOKEN="your-token" \
  -e DISCORD_CHANNEL_ID="your-channel-id" \
  -e DB_HOST="your-db-host" \
  -e DB_PASSWORD="your-password" \
  75-hard-bot
```

### Docker Compose
```bash
cp docker-compose.example.yml docker-compose.yml
# Edit with your tokens, then:
docker-compose up -d
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_BOT_TOKEN` | ✅ Yes | - | Discord bot token |
| `DISCORD_CHANNEL_ID` | ✅ Yes | - | Channel ID where bot operates |
| `DEV_MODE` | ❌ No | `false` | Set to `dev`, `development`, `true`, or `1` to enable dev mode (shows detailed Discord confirmations and DB entries) |
| `LOG_LEVEL` | ❌ No | `ERROR` | Logging verbosity: `INFO` (all logs including DB operations) or `ERROR` (errors only) |
| `DB_HOST` | ❌ No | - | PostgreSQL host (enables database features) |
| `DB_PORT` | ❌ No | `5432` | PostgreSQL port |
| `DB_USER` | ❌ No | `postgres` | Database user |
| `DB_PASSWORD` | ❌ No* | - | Database password (*required if DB_HOST set) |
| `DB_NAME` | ❌ No | `hard75` | Database name |
| `DB_SSLMODE` | ❌ No | `require` | SSL mode (`disable` for local dev) |

## Database Setup

**External Database**: Set `DB_HOST` and `DB_PASSWORD`. Bot auto-runs migrations and creates trigger.

**Docker Compose**: Use `docker-compose.example.yml` for auto-provisioned PostgreSQL.

**Migrations**: Auto-applied on startup. Optional SQL files in `sql/` directory.

## Features & TODOs

### ✅ Implemented
- Daily check-in via ✅ emoji reaction (auto-populates all feats)
- `/exercise quick` - Quick log with defaults
- `/exercise detailed` - Detailed log via modal form
- `/summary [user]` - View progress (all users or specific user)
- Database migrations with checksum validation
- Idempotent tracking (preserves manual entries)

### 🚧 TODO
- [ ] `/diet`, `/water`, `/self-improvement`, `/finances` commands
- [ ] Weekly progress photo reminders
- [ ] Failure tracking (+7 day penalties)
- [ ] Council exception system
- [ ] Custom 75 day start dates/tracking

## Project Structure

```
├── main.go                    # Application entry point
├── Dockerfile                 # Container build config
├── docker-compose.example.yml # Example compose file
├── migrations/                # SQL migrations (auto-applied)
├── sql/                       # Optional SQL files
└── internal/database/         # Database connection & migrations
```
