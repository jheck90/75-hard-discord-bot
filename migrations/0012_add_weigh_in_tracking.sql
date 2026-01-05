-- Migration: 0012_add_weigh_in_tracking
-- Description: Creates table for tracking daily weigh-ins under diet tracking

BEGIN;

CREATE TABLE IF NOT EXISTS weigh_ins (
    weigh_in_id SERIAL PRIMARY KEY,
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    weight_lbs DECIMAL(5,2) NOT NULL,  -- Weight in pounds (supports up to 999.99 lbs)
    weighed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    notes TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (weight_lbs > 0 AND weight_lbs < 1000)  -- Reasonable weight range
);

CREATE INDEX IF NOT EXISTS idx_weigh_ins_user_day 
    ON weigh_ins(user_id, challenge_day);

CREATE INDEX IF NOT EXISTS idx_weigh_ins_user_date 
    ON weigh_ins(user_id, weighed_at);

-- Allow multiple weigh-ins per day (user can weigh in multiple times)
-- But add a unique constraint if you want only one per day per user
-- Uncomment the following if you want to enforce one weigh-in per day:
-- CREATE UNIQUE INDEX IF NOT EXISTS idx_weigh_ins_user_day_unique 
--     ON weigh_ins(user_id, challenge_day);

COMMIT;
