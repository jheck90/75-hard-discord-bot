-- Migration: 0007_add_finances_tracking
-- Description: Creates table for tracking daily spending compliance (necessities only)

BEGIN;

CREATE TABLE IF NOT EXISTS finances_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    compliance_status VARCHAR(20) NOT NULL DEFAULT 'compliant',  -- 'compliant', 'non_compliant'
    -- Track any non-necessity spending (for accountability)
    non_necessity_spending DECIMAL(10, 2) DEFAULT 0,
    spending_category VARCHAR(100),  -- e.g., 'necessity', 'impulse', 'luxury', 'discretionary'
    notes TEXT,
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (compliance_status IN ('compliant', 'non_compliant')),
    CHECK (non_necessity_spending >= 0)
);

CREATE INDEX IF NOT EXISTS idx_finances_completions_user_day 
    ON finances_completions(user_id, challenge_day);

COMMIT;
