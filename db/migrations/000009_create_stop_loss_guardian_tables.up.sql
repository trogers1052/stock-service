-- Stop Loss Guardian Tables
-- Migration: 000009_create_stop_loss_guardian_tables
-- Purpose: Track stop losses and urgent alerts for capital protection

-- Stop loss tracking for all open positions
CREATE TABLE IF NOT EXISTS stop_loss_tracking (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    position_id INTEGER,  -- References journal_positions if available
    robinhood_position_id VARCHAR(50),  -- From Robinhood sync

    -- Position details
    entry_price DECIMAL(10,4) NOT NULL,
    entry_date TIMESTAMP WITH TIME ZONE,
    quantity DECIMAL(18,8) NOT NULL,

    -- Stop loss configuration
    stop_loss_price DECIMAL(10,4),  -- NULL if not set (ALERT!)
    stop_loss_type VARCHAR(20),  -- 'atr_based', 'support', 'percentage', 'manual', NULL
    stop_loss_pct DECIMAL(5,2),  -- % below entry (e.g., 10.00 for 10%)
    stop_loss_set_at TIMESTAMP WITH TIME ZONE,

    -- Current state
    current_price DECIMAL(10,4),
    current_drawdown_pct DECIMAL(5,2),  -- Negative = loss
    price_updated_at TIMESTAMP WITH TIME ZONE,

    -- Alert tracking
    missing_stop_alert_sent BOOLEAN DEFAULT false,
    last_alert_sent TIMESTAMP WITH TIME ZONE,
    alert_count INT DEFAULT 0,
    alert_escalation_level VARCHAR(20) DEFAULT 'none',  -- 'none', 'telegram', 'sms', 'phone_call'

    -- Acknowledgment
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_reason TEXT,

    -- Earnings awareness
    next_earnings_date DATE,
    earnings_alert_sent BOOLEAN DEFAULT false,

    -- Metadata
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_symbol_position UNIQUE (symbol, position_id)
);

-- Index for fast lookups
CREATE INDEX idx_stop_loss_symbol ON stop_loss_tracking(symbol);
CREATE INDEX idx_stop_loss_missing ON stop_loss_tracking(stop_loss_price) WHERE stop_loss_price IS NULL;
CREATE INDEX idx_stop_loss_unacknowledged ON stop_loss_tracking(acknowledged) WHERE acknowledged = false;

-- Urgent alert history for audit trail and escalation
CREATE TABLE IF NOT EXISTS urgent_alerts (
    id SERIAL PRIMARY KEY,

    -- Alert identification
    alert_type VARCHAR(50) NOT NULL,  -- 'missing_stop_loss', 'drawdown_warning', 'drawdown_critical', 'earnings_warning', 'position_size_warning'
    symbol VARCHAR(10),
    position_id INTEGER,
    stop_loss_tracking_id INTEGER REFERENCES stop_loss_tracking(id),

    -- Severity and escalation
    severity VARCHAR(20) NOT NULL,  -- 'info', 'warning', 'urgent', 'critical'
    escalation_level INT DEFAULT 0,  -- 0=telegram, 1=sms, 2=phone_call

    -- Alert content
    message TEXT NOT NULL,
    details JSONB,  -- Additional context: { entry_price, current_price, drawdown_pct, suggested_stop }

    -- Delivery tracking
    channel VARCHAR(20) NOT NULL,  -- 'telegram', 'sms', 'phone_call'
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivery_status VARCHAR(20) DEFAULT 'sent',  -- 'sent', 'delivered', 'failed'
    twilio_sid VARCHAR(50),  -- Twilio message/call SID for tracking

    -- Acknowledgment
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_via VARCHAR(20),  -- 'telegram_reply', 'sms_reply', 'manual'

    -- Escalation tracking
    escalated_from_id INTEGER REFERENCES urgent_alerts(id),
    next_escalation_at TIMESTAMP WITH TIME ZONE,
    max_escalation_reached BOOLEAN DEFAULT false,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for alert queries
CREATE INDEX idx_urgent_alerts_type ON urgent_alerts(alert_type);
CREATE INDEX idx_urgent_alerts_symbol ON urgent_alerts(symbol);
CREATE INDEX idx_urgent_alerts_unack ON urgent_alerts(acknowledged) WHERE acknowledged = false;
CREATE INDEX idx_urgent_alerts_escalation ON urgent_alerts(next_escalation_at) WHERE next_escalation_at IS NOT NULL;

-- Behavioral patterns for Trade Psychologist (Phase 3)
CREATE TABLE IF NOT EXISTS behavioral_patterns (
    id SERIAL PRIMARY KEY,
    pattern_type VARCHAR(50) NOT NULL,  -- 'fomo', 'revenge_trade', 'hope_hold', 'overconfidence', 'sector_fixation', 'monday_overtrading'
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Links
    trade_id INTEGER,
    position_id INTEGER,
    journal_entry_id INTEGER,

    -- Pattern details
    severity VARCHAR(20) NOT NULL,  -- 'info', 'warning', 'critical'
    confidence DECIMAL(5,4),  -- 0.0000 to 1.0000
    evidence JSONB NOT NULL,  -- Detailed evidence supporting detection

    -- Intervention
    intervention_sent BOOLEAN DEFAULT false,
    intervention_sent_at TIMESTAMP WITH TIME ZONE,
    intervention_message TEXT,
    intervention_acknowledged BOOLEAN DEFAULT false,
    intervention_acknowledged_at TIMESTAMP WITH TIME ZONE,

    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_behavioral_patterns_type ON behavioral_patterns(pattern_type);
CREATE INDEX idx_behavioral_patterns_position ON behavioral_patterns(position_id);

-- Trade checklists for pre-trade validation
CREATE TABLE IF NOT EXISTS trade_checklists (
    id SERIAL PRIMARY KEY,

    -- Trade identification
    trade_id INTEGER,
    position_id INTEGER,
    symbol VARCHAR(10) NOT NULL,

    -- Checklist items
    stop_loss_defined BOOLEAN DEFAULT false,
    stop_loss_price DECIMAL(10,4),
    position_sized_correctly BOOLEAN DEFAULT false,
    position_size_shares INT,
    position_size_pct DECIMAL(5,2),  -- % of portfolio
    risk_per_trade_pct DECIMAL(5,2),  -- % of portfolio at risk
    rr_ratio DECIMAL(5,2),  -- Reward:Risk ratio
    rr_ratio_acceptable BOOLEAN DEFAULT false,  -- >= 2.0
    no_earnings_imminent BOOLEAN DEFAULT false,
    days_to_earnings INT,
    regime_compatible BOOLEAN DEFAULT false,
    regime_id VARCHAR(30),

    -- Result
    all_checks_passed BOOLEAN DEFAULT false,
    checks_failed TEXT[],  -- List of failed check names
    override_reason TEXT,  -- If trader proceeded despite failed checks

    checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_trade_checklists_symbol ON trade_checklists(symbol);
CREATE INDEX idx_trade_checklists_position ON trade_checklists(position_id);

-- Function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to stop_loss_tracking
DROP TRIGGER IF EXISTS update_stop_loss_tracking_updated_at ON stop_loss_tracking;
CREATE TRIGGER update_stop_loss_tracking_updated_at
    BEFORE UPDATE ON stop_loss_tracking
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
