-- Store trade plan with each signal for outcome tracking.
-- Enables: did price hit target or stop after the signal fired?

ALTER TABLE signal_feedback
    ADD COLUMN entry_price DECIMAL(10,4),
    ADD COLUMN stop_price DECIMAL(10,4),
    ADD COLUMN target_1 DECIMAL(10,4),
    ADD COLUMN target_2 DECIMAL(10,4),
    ADD COLUMN valid_until TIMESTAMP WITH TIME ZONE;
