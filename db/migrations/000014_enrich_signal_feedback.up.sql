-- Enrich signal_feedback with rule + regime data for per-rule accuracy tracking.
-- Enables the feedback loop: signal quality data flows back to decision-engine.

ALTER TABLE signal_feedback
    ADD COLUMN rules_triggered TEXT[],
    ADD COLUMN regime_id VARCHAR(30),
    ADD COLUMN decision_confidence DECIMAL(5,4);

CREATE INDEX idx_signal_feedback_regime ON signal_feedback(regime_id);
CREATE INDEX idx_signal_feedback_rules ON signal_feedback USING GIN(rules_triggered);
