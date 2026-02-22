CREATE TABLE IF NOT EXISTS signal_feedback (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    signal VARCHAR(10) NOT NULL,
    action VARCHAR(10) NOT NULL,
    confidence DECIMAL(5,4),
    feedback_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_signal_feedback_symbol ON signal_feedback(symbol);
CREATE INDEX idx_signal_feedback_action ON signal_feedback(action);
CREATE INDEX idx_signal_feedback_timestamp ON signal_feedback(feedback_timestamp);
