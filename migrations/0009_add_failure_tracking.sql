-- Migration: 0009_add_failure_tracking
-- Description: Creates tables for tracking missed days, failures, and council exceptions

BEGIN;

-- Track days where user failed to complete all requirements
CREATE TABLE IF NOT EXISTS challenge_failures (
    failure_id SERIAL PRIMARY KEY,
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    failed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    failed_feats TEXT[],  -- Array of feats that were failed: ['exercise', 'diet', 'water', etc.]
    days_added INTEGER DEFAULT 7,  -- Days added to challenge (default 7, can be waived by council)
    council_forgiven BOOLEAN DEFAULT false,
    council_forgiven_at TIMESTAMP WITH TIME ZONE,
    council_forgiven_by VARCHAR(20),  -- User ID of council member who approved
    notes TEXT,
    metadata JSONB,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (days_added >= 0),
    UNIQUE(user_id, challenge_day)
);

CREATE INDEX IF NOT EXISTS idx_challenge_failures_user_day 
    ON challenge_failures(user_id, challenge_day);

CREATE INDEX IF NOT EXISTS idx_challenge_failures_forgiven 
    ON challenge_failures(council_forgiven);

-- Track council exception approvals (for audit trail)
CREATE TABLE IF NOT EXISTS council_exceptions (
    exception_id SERIAL PRIMARY KEY,
    failure_id INTEGER NOT NULL REFERENCES challenge_failures(failure_id) ON DELETE CASCADE,
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_by VARCHAR(20) NOT NULL,  -- User ID of council member
    reason TEXT NOT NULL,  -- Reason for exception (illness, injury, family emergency, etc.)
    approved_within_24h BOOLEAN NOT NULL,  -- Must be approved within 24 hours
    metadata JSONB,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (approved_at <= requested_at + INTERVAL '24 hours')
);

CREATE INDEX IF NOT EXISTS idx_council_exceptions_user_day 
    ON council_exceptions(user_id, challenge_day);

CREATE INDEX IF NOT EXISTS idx_council_exceptions_approved_by 
    ON council_exceptions(approved_by);

COMMIT;
