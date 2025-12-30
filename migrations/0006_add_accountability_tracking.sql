-- Migration: 0006_add_accountability_tracking
-- Description: Creates tables for tracking accountability (daily check-in + weekly progress photo)

BEGIN;

-- Daily check-in tracking (primary accountability mechanism)
-- When a user checks in, this triggers automatic population of all feat tables
CREATE TABLE IF NOT EXISTS accountability_checkins (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    check_in_method VARCHAR(50) DEFAULT 'emoji_reaction',  -- e.g., 'emoji_reaction', 'slash_command', 'message'
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1)
);

CREATE INDEX IF NOT EXISTS idx_accountability_checkins_user_day 
    ON accountability_checkins(user_id, challenge_day);

CREATE INDEX IF NOT EXISTS idx_accountability_checkins_date 
    ON accountability_checkins(completed_at);

-- Weekly progress photo tracking (once per week)
CREATE TABLE IF NOT EXISTS progress_photos (
    user_id VARCHAR(20) NOT NULL,
    challenge_week INTEGER NOT NULL,  -- Week number (1+ for standard 75 days, can extend)
    challenge_day INTEGER NOT NULL,    -- Day of the week when photo was taken
    photo_taken_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    photo_url TEXT,
    photo_storage_key VARCHAR(255),
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_week),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_week >= 1),
    CHECK (challenge_day >= 1)
);

CREATE INDEX IF NOT EXISTS idx_progress_photos_user_week 
    ON progress_photos(user_id, challenge_week);

COMMIT;
