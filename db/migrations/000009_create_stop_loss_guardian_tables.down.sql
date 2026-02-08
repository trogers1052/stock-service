-- Rollback: Stop Loss Guardian Tables

DROP TRIGGER IF EXISTS update_stop_loss_tracking_updated_at ON stop_loss_tracking;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS trade_checklists;
DROP TABLE IF EXISTS behavioral_patterns;
DROP TABLE IF EXISTS urgent_alerts;
DROP TABLE IF EXISTS stop_loss_tracking;
