-- Migration: Create backtest_tiers table for tier ranking metadata
-- Stores composite scores, validation gate results, regime conditions,
-- and pre-computed multipliers from backtesting validation reports.

CREATE TABLE IF NOT EXISTS backtest_tiers (
    symbol                   VARCHAR(10) PRIMARY KEY REFERENCES stocks(symbol) ON DELETE CASCADE,
    tier                     VARCHAR(1) NOT NULL CHECK (tier IN ('S','A','B','C','D','F')),
    composite_score          DECIMAL(5,1) NOT NULL CHECK (composite_score >= 0 AND composite_score <= 100),
    gates_passed             INTEGER NOT NULL DEFAULT 0 CHECK (gates_passed >= 0 AND gates_passed <= 4),
    gates_total              INTEGER NOT NULL DEFAULT 4,
    regime_pass              BOOLEAN NOT NULL DEFAULT false,
    allowed_regimes          TEXT[],
    sharpe                   DECIMAL(6,2),
    total_return             DECIMAL(8,1),
    win_rate                 DECIMAL(5,1),
    profit_factor            DECIMAL(8,2),
    max_drawdown             DECIMAL(5,1),
    trade_count              INTEGER DEFAULT 0,
    confidence_multiplier    DECIMAL(4,2) NOT NULL DEFAULT 1.00,
    position_size_multiplier DECIMAL(4,2) NOT NULL DEFAULT 1.00,
    blacklisted              BOOLEAN NOT NULL DEFAULT false,
    ranking_date             DATE NOT NULL,
    notes                    TEXT,
    created_at               TIMESTAMPTZ DEFAULT NOW(),
    updated_at               TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_backtest_tiers_tier ON backtest_tiers(tier);
CREATE INDEX idx_backtest_tiers_score ON backtest_tiers(composite_score DESC);
