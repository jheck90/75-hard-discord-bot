-- Migration: 0003_add_diet_tracking
-- Description: Creates table for tracking daily diet compliance (one defined diet, no cheat meals, no alcohol)

BEGIN;

CREATE TABLE IF NOT EXISTS diet_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    diet_type VARCHAR(100),  -- User's chosen diet (e.g., 'keto', 'paleo', 'calorie_deficit', etc.)
    cheat_meal BOOLEAN DEFAULT false,  -- Must be false for compliance
    alcohol_consumed BOOLEAN DEFAULT false,  -- Must be false for compliance
    notes TEXT,
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (cheat_meal = false),  -- Enforce no cheat meals
    CHECK (alcohol_consumed = false)  -- Enforce no alcohol
);

CREATE INDEX IF NOT EXISTS idx_diet_completions_user_day 
    ON diet_completions(user_id, challenge_day);

COMMIT;
