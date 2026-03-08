DROP INDEX IF EXISTS idx_signal_feedback_rules;
DROP INDEX IF EXISTS idx_signal_feedback_regime;

ALTER TABLE signal_feedback
    DROP COLUMN IF EXISTS decision_confidence,
    DROP COLUMN IF EXISTS regime_id,
    DROP COLUMN IF EXISTS rules_triggered;
