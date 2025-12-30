# 75 Hard Discord Bot - High Level Overview

## Core Concept

Discord-first accountability bot for the 75 Hard challenge. The entire purpose is accountability on a semi-public forum. Users opt-in to the challenge and are invited to a sub-channel where the bot lives and posts reports.

## Architecture Philosophy

- **Discord-first**: Primary interface is Discord, web app is a future feature
- **Database-driven**: All data is managed by PostgreSQL and manipulated by the bot
- **Modular design**: Functions should be modular to support adding features later
- **Docker deployment**: Must run as a Docker container

## Technology Stack

- **Language**: Golang
- **Database**: PostgreSQL backend
- **Deployment**: Docker container

## Database Configuration

Supports two operational modes:
1. **External Database**: Use provided PostgreSQL connection details (host, port, user, password, database name)
2. **Auto-provisioned Database**: If no database connection details are provided, automatically spin up a PostgreSQL instance (likely as a sidecar container or embedded)

The system should handle database initialization/migration automatically.

## Core Functionality

### Daily Check-in System

- **Daily Prompt**: Bot prompts users daily with: "Did you complete your 75 hard today?"
- **User Response**: Users react with a check emoji (✅) to indicate completion
- **No Action Policy**: If users do not add the check emoji, take no action (no tracking, no reminders, etc.)
- **Tracking**: Track which users have completed their daily check-in (stored in PostgreSQL)
- **Duration**: Users are required to input daily over the course of 75 days

### Slash Commands

- **Implementation**: Regex-friendly slash commands for flexibility
- **Initial Commands**:
  - `/s` or `/summarize`: Displays a Discord message-friendly chart showing users and how many days they've completed
    - Chart format should be suitable for Discord message display (consider using code blocks, tables, or formatted text)

## User Flow

1. Users opt-in to the 75 Hard challenge
2. Users are invited to a sub-channel where the bot operates
3. Bot posts daily check-in prompts
4. Users react with ✅ to log completion
5. Bot tracks progress in PostgreSQL
6. Users can view summaries via slash commands

## Future Considerations

- Web app as a feature (data remains managed by PostgreSQL)
- Additional slash commands and features
- Reporting and analytics features
- Modular architecture supports easy feature additions
