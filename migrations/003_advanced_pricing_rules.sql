-- Add advanced pricing rule types to vendor_pricing_rules
ALTER TABLE vendor_pricing_rules ADD COLUMN IF NOT EXISTS weekend_premium_type TEXT DEFAULT 'percentage';
ALTER TABLE vendor_pricing_rules ADD COLUMN IF NOT EXISTS last_minute_booking_type TEXT DEFAULT 'percentage';

-- Ensure types follow a convention
-- CHECK constraints if needed, but keeping it flexible for now
