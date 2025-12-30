-- Migration: 0002_add_exercise_tracking
-- Description: Creates table for tracking daily movement (workout + core/mobility)

BEGIN;

CREATE TABLE IF NOT EXISTS exercise_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Main workout (30+ minutes required)
    workout_duration_minutes INTEGER DEFAULT 30,  -- Default minimum requirement
    workout_type VARCHAR(100) DEFAULT 'general',  -- Default type
    workout_location VARCHAR(50) DEFAULT 'indoor',  -- Default location
    weight_vest_used BOOLEAN DEFAULT false,  -- Required if walking
    -- Core/mobility (10 minutes required)
    core_mobility_duration_minutes INTEGER DEFAULT 10,  -- Default minimum requirement
    core_mobility_type VARCHAR(100) DEFAULT 'general',  -- Default type
    notes TEXT,
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (workout_duration_minutes IS NULL OR workout_duration_minutes >= 30),
    CHECK (core_mobility_duration_minutes IS NULL OR core_mobility_duration_minutes >= 10)
);

CREATE INDEX IF NOT EXISTS idx_exercise_completions_user_day 
    ON exercise_completions(user_id, challenge_day);

COMMIT;
