# 75-hard-discord-bot
A discord bot for managing the 75 Hard Challenge.

## Development

### Prerequisites
- Go 1.21 or later
- Docker (for containerized deployment)
- Discord webhook URL (stored in `.envrc`)
- PostgreSQL database (optional - see Database Configuration below)

### Database Configuration

The bot supports two database modes:

#### Option 1: External PostgreSQL Database

Provide your own PostgreSQL database connection details via environment variables:

```bash
export DB_HOST=your-db-host
export DB_PORT=5432                    # Optional, defaults to 5432
export DB_USER=postgres                 # Optional, defaults to postgres
export DB_PASSWORD=your-password        # Required if DB_HOST is set
export DB_NAME=hard75                   # Optional, defaults to hard75
export DB_SSLMODE=require              # Optional, defaults to require (use 'disable' for local dev)
```

The bot will automatically:
- Connect to your database on startup
- Run all pending migrations
- Validate existing migrations haven't been corrupted

#### Option 2: Auto-Provisioned Database (Docker Compose)

If no database connection details are provided, the bot can run with an auto-provisioned PostgreSQL instance using Docker Compose. See the [Docker Deployment](#docker-deployment) section below.

**Note**: The bot can run without a database for basic webhook functionality, but full challenge tracking features require a database connection.

### Local Testing

1. Load environment variables (if using direnv):
   ```bash
   direnv allow
   ```
   Or manually source:
   ```bash
   source .envrc
   ```

2. (Optional) Set up database environment variables if using an external database:
   ```bash
   export DB_HOST=localhost
   export DB_PORT=5432
   export DB_USER=postgres
   export DB_PASSWORD=your-password
   export DB_NAME=hard75
   ```

3. Run the bot:
   ```bash
   go run main.go
   ```

   This will:
   - Connect to database (if configured) and run migrations
   - Send a test "ping" message to your Discord webhook

4. Test emoji reaction feature:
   ```bash
   go run main.go test-emoji
   ```

   This requires:
   - `DISCORD_BOT_TOKEN` - Your Discord bot token (create at https://discord.com/developers/applications)
   - `DISCORD_CHANNEL_ID` - The channel ID where you want to test
   
   The bot will:
   - Send a message saying "emoji this message to ping"
   - Listen for emoji reactions on that message
   - Respond with a confirmation showing the user's name and the emoji they added

### Docker Deployment

#### Option 1: With External Database

1. Build the Docker image:
   ```bash
   docker build -t 75-hard-bot .
   ```

2. Run the container with database connection details:
   ```bash
   docker run \
     -e DISCORD_WEBHOOK_URL="your-webhook-url" \
     -e DB_HOST="your-db-host" \
     -e DB_PORT="5432" \
     -e DB_USER="postgres" \
     -e DB_PASSWORD="your-password" \
     -e DB_NAME="hard75" \
     -e DB_SSLMODE="require" \
     75-hard-bot
   ```
   
   **Note**: SSL is enabled by default (`require`). Use `DB_SSLMODE="disable"` only for local development or if your database doesn't support SSL.

#### Option 2: Auto-Provisioned Database (Docker Compose)

Copy the example docker-compose file and customize:
```bash
cp docker-compose.example.yml docker-compose.yml
# Edit docker-compose.yml with your DISCORD_WEBHOOK_URL
```

Or create a `docker-compose.yml` file manually:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: hard75
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  bot:
    build: .
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DISCORD_WEBHOOK_URL: "your-webhook-url"
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: hard75
      DB_SSLMODE: disable  # Use 'disable' for local Docker Compose, 'require' for production
    restart: unless-stopped

volumes:
  postgres_data:
```

Then run:
```bash
docker-compose up -d
```

The bot will automatically:
- Wait for PostgreSQL to be ready
- Connect and run migrations on first startup
- Start tracking challenge data

#### Option 3: Webhook-Only Mode (No Database)

Run without database configuration for basic webhook functionality:

```bash
docker run -e DISCORD_WEBHOOK_URL="your-webhook-url" 75-hard-bot
```

The bot will run in webhook-only mode without database features.

## Project Structure

- `main.go` - Main application entry point
- `Dockerfile` - Container build configuration
- `docker-compose.example.yml` - Example Docker Compose configuration for auto-provisioned database
- `migrations/` - SQL migration files for database schema (auto-applied)
- `sql/` - Optional SQL files for functions and views (manual application)
- `internal/database/` - Database connection and migration management
- `.envrc` - Environment variables (not committed to git)
- `vibe-code-resources/` - Project documentation and context

## Database Migrations

The bot uses an automatic migration system that:
- Tracks migration versions in the `schema_migrations` table
- Validates migration integrity using SHA-256 checksums
- Applies pending migrations automatically on startup
- Prevents manual database modifications from corrupting the schema

All migrations are stored in the `migrations/` directory and follow the naming convention: `NNNN_description.sql`

### Optional Database Objects

Some database objects (functions, views) are stored separately in the `sql/` directory and can be applied manually:

- **`sql/auto_populate_trigger.sql`** - Creates the trigger function that auto-populates feat tables when users check in
- **`sql/summary_view.sql`** - Creates a materialized view for quick progress queries

To apply these manually:
```bash
psql -h your-db-host -U postgres -d hard75 -f sql/auto_populate_trigger.sql
psql -h your-db-host -U postgres -d hard75 -f sql/summary_view.sql
```

Or via Docker:
```bash
docker exec -i your-postgres-container psql -U postgres -d hard75 < sql/auto_populate_trigger.sql
docker exec -i your-postgres-container psql -U postgres -d hard75 < sql/summary_view.sql
```

**Note**: These are optional - the bot works fine without them. The trigger provides convenience (auto-populating feat tables), and the view provides faster query performance.
