-- Migration: 0004_add_water_tracking
-- Description: Creates table for tracking daily water intake (1 gallon plain water required)

BEGIN;

CREATE TABLE IF NOT EXISTS water_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    amount_ounces DECIMAL(5, 2) DEFAULT 128.00,  -- Default: 1 gallon (128 ounces)
    is_plain_water BOOLEAN DEFAULT true,  -- Must be plain water (no additives, flavoring, etc.)
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (amount_ounces IS NULL OR amount_ounces >= 0),
    CHECK (is_plain_water = true)  -- Enforce plain water only
);

CREATE INDEX IF NOT EXISTS idx_water_completions_user_day 
    ON water_completions(user_id, challenge_day);

COMMIT;
