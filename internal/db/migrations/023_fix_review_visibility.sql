-- Migration: 023_fix_review_visibility
-- Description: Fix NULL values in vendor_reviews for newly added columns

-- Safety check: ensure columns exist (should be added by 022, but let's be safe)
ALTER TABLE vendor_reviews ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT TRUE;
ALTER TABLE vendor_reviews ADD COLUMN IF NOT EXISTS helpful_count INTEGER DEFAULT 0;

-- Update existing NULL values
UPDATE vendor_reviews SET is_public = TRUE WHERE is_public IS NULL;
UPDATE vendor_reviews SET helpful_count = 0 WHERE helpful_count IS NULL;

-- Ensure defaults are consistently applied for future rows
ALTER TABLE vendor_reviews ALTER COLUMN is_public SET DEFAULT TRUE;
ALTER TABLE vendor_reviews ALTER COLUMN helpful_count SET DEFAULT 0;
