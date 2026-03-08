DROP INDEX IF EXISTS idx_signal_feedback_unresolved;

ALTER TABLE signal_feedback
    DROP COLUMN IF EXISTS outcome_at,
    DROP COLUMN IF EXISTS outcome;
