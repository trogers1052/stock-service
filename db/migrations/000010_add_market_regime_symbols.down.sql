-- Remove market regime and sector ETFs

-- Remove from monitored_stocks first (FK constraint)
DELETE FROM monitored_stocks
WHERE symbol IN ('SPY', 'QQQ', 'XLK', 'XLF', 'XLE', 'XLV', 'XLI', 'XLY', 'XLP', 'XLU');

-- Remove from stocks table
DELETE FROM stocks
WHERE symbol IN ('SPY', 'QQQ', 'XLK', 'XLF', 'XLE', 'XLV', 'XLI', 'XLY', 'XLP', 'XLU');
