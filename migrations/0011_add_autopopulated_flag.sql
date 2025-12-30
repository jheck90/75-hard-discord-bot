-- Migration: 0011_add_autopopulated_flag
-- Description: Adds autopopulated boolean flag to all feat tables to track auto vs manual entries

BEGIN;

-- Add autopopulated flag to exercise_completions
ALTER TABLE exercise_completions 
ADD COLUMN IF NOT EXISTS autopopulated BOOLEAN DEFAULT true;

-- Add autopopulated flag to diet_completions
ALTER TABLE diet_completions 
ADD COLUMN IF NOT EXISTS autopopulated BOOLEAN DEFAULT true;

-- Add autopopulated flag to water_completions
ALTER TABLE water_completions 
ADD COLUMN IF NOT EXISTS autopopulated BOOLEAN DEFAULT true;

-- Add autopopulated flag to self_improvement_completions
ALTER TABLE self_improvement_completions 
ADD COLUMN IF NOT EXISTS autopopulated BOOLEAN DEFAULT true;

-- Add autopopulated flag to finances_completions
ALTER TABLE finances_completions 
ADD COLUMN IF NOT EXISTS autopopulated BOOLEAN DEFAULT true;

-- Set existing records to autopopulated = false (they were likely manual entries)
UPDATE exercise_completions SET autopopulated = false WHERE autopopulated IS NULL;
UPDATE diet_completions SET autopopulated = false WHERE autopopulated IS NULL;
UPDATE water_completions SET autopopulated = false WHERE autopopulated IS NULL;
UPDATE self_improvement_completions SET autopopulated = false WHERE autopopulated IS NULL;
UPDATE finances_completions SET autopopulated = false WHERE autopopulated IS NULL;

COMMIT;
