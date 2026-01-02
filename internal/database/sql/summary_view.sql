-- Summary view for quick progress queries
-- This can be applied manually or via a separate command
-- Run this after all table migrations are complete

CREATE MATERIALIZED VIEW IF NOT EXISTS user_daily_progress AS
SELECT 
    u.user_id,
    u.username,
    u.challenge_start_date,
    u.current_challenge_end_date,
    u.days_added,
    day.day_number,
    CASE WHEN e.user_id IS NOT NULL THEN true ELSE false END AS exercise_completed,
    CASE WHEN d.user_id IS NOT NULL THEN true ELSE false END AS diet_completed,
    CASE WHEN w.user_id IS NOT NULL THEN true ELSE false END AS water_completed,
    CASE WHEN si.user_id IS NOT NULL THEN true ELSE false END AS self_improvement_completed,
    CASE WHEN a.user_id IS NOT NULL THEN true ELSE false END AS accountability_checkin_completed,
    CASE WHEN f.user_id IS NOT NULL AND f.compliance_status = 'compliant' THEN true ELSE false END AS finances_completed,
    -- Check if day was failed
    CASE WHEN cf.failure_id IS NOT NULL THEN true ELSE false END AS day_failed,
    CASE WHEN cf.council_forgiven THEN true ELSE false END AS failure_forgiven,
    -- Overall completion: all feats completed for the day
    CASE WHEN 
        e.user_id IS NOT NULL AND
        d.user_id IS NOT NULL AND
        w.user_id IS NOT NULL AND
        si.user_id IS NOT NULL AND
        a.user_id IS NOT NULL AND
        f.user_id IS NOT NULL AND
        f.compliance_status = 'compliant' AND
        cf.failure_id IS NULL
    THEN true ELSE false END AS all_feats_completed
FROM users u
CROSS JOIN generate_series(1, (SELECT MAX(current_challenge_end_date - challenge_start_date) FROM users)) AS day(day_number)
LEFT JOIN exercise_completions e ON e.user_id = u.user_id AND e.challenge_day = day.day_number
LEFT JOIN diet_completions d ON d.user_id = u.user_id AND d.challenge_day = day.day_number
LEFT JOIN water_completions w ON w.user_id = u.user_id AND w.challenge_day = day.day_number
LEFT JOIN self_improvement_completions si ON si.user_id = u.user_id AND si.challenge_day = day.day_number
LEFT JOIN accountability_checkins a ON a.user_id = u.user_id AND a.challenge_day = day.day_number
LEFT JOIN finances_completions f ON f.user_id = u.user_id AND f.challenge_day = day.day_number
LEFT JOIN challenge_failures cf ON cf.user_id = u.user_id AND cf.challenge_day = day.day_number
WHERE day.day_number <= (u.current_challenge_end_date - u.challenge_start_date);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_daily_progress_user_day 
    ON user_daily_progress(user_id, day_number);

-- Refresh function for the materialized view
CREATE OR REPLACE FUNCTION refresh_user_daily_progress()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_daily_progress;
END;
$$ LANGUAGE plpgsql;
