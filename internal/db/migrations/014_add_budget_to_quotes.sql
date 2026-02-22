-- Up Migration
ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS budget_range TEXT;

-- Down Migration
-- ALTER TABLE quote_requests DROP COLUMN IF EXISTS budget_range;
