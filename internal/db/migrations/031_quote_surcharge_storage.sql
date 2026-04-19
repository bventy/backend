-- Migration: 031_quote_surcharge_storage
-- Description: Store calculated surcharge information in quote requests for historical accuracy.

-- Add surcharge_details JSONB to store breakdown
-- Add is_premium boolean for quick identification
ALTER TABLE "quote_requests" 
ADD COLUMN IF NOT EXISTS "surcharge_details" JSONB,
ADD COLUMN IF NOT EXISTS "is_premium" BOOLEAN DEFAULT false;

-- Add index for premium filtering
CREATE INDEX IF NOT EXISTS idx_quote_requests_premium ON quote_requests(is_premium) WHERE is_premium = true;
