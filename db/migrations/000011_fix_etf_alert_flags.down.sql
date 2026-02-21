-- Revert: restore default alert flags for regime/sector ETFs

UPDATE monitored_stocks
SET alert_on_buy_zone    = TRUE,
    alert_on_rsi_oversold = FALSE,
    updated_at            = NOW()
WHERE symbol IN ('SPY', 'QQQ', 'XLK', 'XLF', 'XLE', 'XLV', 'XLI', 'XLY', 'XLP', 'XLU');
