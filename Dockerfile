# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code and migrations
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bot .

# Copy migrations directory (needed at runtime)
COPY --from=builder /app/migrations ./migrations

# Environment variables documentation (set via docker run or docker-compose)
# Required:
#   DISCORD_BOT_TOKEN - Discord bot token
#   DISCORD_CHANNEL_ID - Channel ID where bot operates
# Optional (for database):
#   DB_HOST - Database host
#   DB_PORT - Database port (default: 5432)
#   DB_USER - Database user (default: postgres)
#   DB_PASSWORD - Database password
#   DB_NAME - Database name (default: hard75)
#   DB_SSLMODE - SSL mode (default: require)

# Expose no ports (bot uses Discord websocket)

# Run the bot
CMD ["./bot"]
