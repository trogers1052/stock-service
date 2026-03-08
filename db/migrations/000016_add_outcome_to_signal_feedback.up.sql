ALTER TABLE signal_feedback
    ADD COLUMN outcome VARCHAR(20),
    ADD COLUMN outcome_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX idx_signal_feedback_unresolved
    ON signal_feedback (id)
    WHERE outcome IS NULL
      AND entry_price IS NOT NULL
      AND stop_price IS NOT NULL;
