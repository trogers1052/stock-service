-- Add market regime and sector ETFs for context-service
-- These are used for regime detection (BULL/BEAR/SIDEWAYS) and sector strength analysis

-- Insert into stocks table first (required by monitored_stocks FK)
INSERT INTO stocks (symbol, name, exchange, sector, created_at, last_updated)
VALUES
    -- Market regime ETFs
    ('SPY', 'SPDR S&P 500 ETF Trust', 'NYSE', 'ETF - Index', NOW(), NOW()),
    ('QQQ', 'Invesco QQQ Trust', 'NASDAQ', 'ETF - Index', NOW(), NOW()),

    -- Sector ETFs
    ('XLK', 'Technology Select Sector SPDR Fund', 'NYSE', 'ETF - Technology', NOW(), NOW()),
    ('XLF', 'Financial Select Sector SPDR Fund', 'NYSE', 'ETF - Financials', NOW(), NOW()),
    ('XLE', 'Energy Select Sector SPDR Fund', 'NYSE', 'ETF - Energy', NOW(), NOW()),
    ('XLV', 'Health Care Select Sector SPDR Fund', 'NYSE', 'ETF - Healthcare', NOW(), NOW()),
    ('XLI', 'Industrial Select Sector SPDR Fund', 'NYSE', 'ETF - Industrials', NOW(), NOW()),
    ('XLY', 'Consumer Discretionary Select Sector SPDR Fund', 'NYSE', 'ETF - Consumer Discretionary', NOW(), NOW()),
    ('XLP', 'Consumer Staples Select Sector SPDR Fund', 'NYSE', 'ETF - Consumer Staples', NOW(), NOW()),
    ('XLU', 'Utilities Select Sector SPDR Fund', 'NYSE', 'ETF - Utilities', NOW(), NOW())
ON CONFLICT (symbol) DO UPDATE SET
    name = EXCLUDED.name,
    sector = EXCLUDED.sector,
    last_updated = NOW();

-- Add to monitored_stocks for data ingestion
INSERT INTO monitored_stocks (symbol, enabled, priority, reason, added_at)
VALUES
    ('SPY', TRUE, 1, 'Market regime detection - S&P 500 index', NOW()),
    ('QQQ', TRUE, 1, 'Market regime detection - Nasdaq 100 index', NOW()),
    ('XLK', TRUE, 2, 'Sector strength - Technology', NOW()),
    ('XLF', TRUE, 2, 'Sector strength - Financials', NOW()),
    ('XLE', TRUE, 2, 'Sector strength - Energy', NOW()),
    ('XLV', TRUE, 2, 'Sector strength - Healthcare', NOW()),
    ('XLI', TRUE, 2, 'Sector strength - Industrials', NOW()),
    ('XLY', TRUE, 2, 'Sector strength - Consumer Discretionary', NOW()),
    ('XLP', TRUE, 2, 'Sector strength - Consumer Staples', NOW()),
    ('XLU', TRUE, 2, 'Sector strength - Utilities', NOW())
ON CONFLICT (symbol) DO UPDATE SET
    enabled = TRUE,
    reason = EXCLUDED.reason,
    updated_at = NOW();
