-- Migration: 0001_initial_schema
-- Description: Creates initial schema with users table and schema_migrations tracking

BEGIN;

-- Schema migrations tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    checksum VARCHAR(64) NOT NULL,
    PRIMARY KEY (version, name)
);

CREATE INDEX IF NOT EXISTS idx_schema_migrations_version 
    ON schema_migrations(version);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(20) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    discriminator VARCHAR(4),
    avatar_url TEXT,
    challenge_start_date DATE NOT NULL,
    original_challenge_end_date DATE NOT NULL,  -- Original 75-day end date
    current_challenge_end_date DATE NOT NULL,   -- Current end date (may be extended due to failures)
    days_added INTEGER DEFAULT 0,               -- Total days added due to failures
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (current_challenge_end_date >= original_challenge_end_date),
    CHECK (original_challenge_end_date > challenge_start_date)
);

CREATE INDEX IF NOT EXISTS idx_users_challenge_dates 
    ON users(challenge_start_date, current_challenge_end_date);

COMMIT;
