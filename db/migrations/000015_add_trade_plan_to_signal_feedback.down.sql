ALTER TABLE signal_feedback
    DROP COLUMN IF EXISTS valid_until,
    DROP COLUMN IF EXISTS target_2,
    DROP COLUMN IF EXISTS target_1,
    DROP COLUMN IF EXISTS stop_price,
    DROP COLUMN IF EXISTS entry_price;
