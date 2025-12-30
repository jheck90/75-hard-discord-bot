-- Auto-populate trigger function
-- This can be applied manually or via a separate command
-- Run this after all table migrations are complete

-- Function to auto-populate all feat tables when accountability check-in is recorded
CREATE OR REPLACE FUNCTION auto_populate_feats_on_checkin()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert exercise completion with defaults
    INSERT INTO exercise_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert diet completion with defaults
    INSERT INTO diet_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert water completion with defaults
    INSERT INTO water_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert self-improvement completion with defaults
    INSERT INTO self_improvement_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert finances completion with defaults
    INSERT INTO finances_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger that fires after insert on accountability_checkins
CREATE TRIGGER trigger_auto_populate_feats
    AFTER INSERT ON accountability_checkins
    FOR EACH ROW
    EXECUTE FUNCTION auto_populate_feats_on_checkin();
