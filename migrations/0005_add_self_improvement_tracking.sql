-- Migration: 0005_add_self_improvement_tracking
-- Description: Creates table for tracking daily self-improvement (30 minutes required)

BEGIN;

CREATE TABLE IF NOT EXISTS self_improvement_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    duration_minutes INTEGER NOT NULL DEFAULT 30,  -- Default minimum requirement
    activity_type VARCHAR(100) DEFAULT 'general',  -- Default type
    description TEXT,
    completed_before_bed BOOLEAN DEFAULT false,  -- If true, must have zero screen time
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (duration_minutes >= 30)
);

CREATE INDEX IF NOT EXISTS idx_self_improvement_completions_user_day 
    ON self_improvement_completions(user_id, challenge_day);

COMMIT;
