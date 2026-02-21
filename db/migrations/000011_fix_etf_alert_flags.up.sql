-- ETFs added in migration 000010 are regime/sector detection instruments,
-- not tradeable positions. Disable buy-zone and RSI alerts so they don't
-- trigger the Alert Dispatcher or Stop Loss Guardian as if they were stocks.

UPDATE monitored_stocks
SET alert_on_buy_zone    = FALSE,
    alert_on_rsi_oversold = FALSE,
    updated_at            = NOW()
WHERE symbol IN ('SPY', 'QQQ', 'XLK', 'XLF', 'XLE', 'XLV', 'XLI', 'XLY', 'XLP', 'XLU');
