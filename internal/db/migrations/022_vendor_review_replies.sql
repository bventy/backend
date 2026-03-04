-- Migration: 022_vendor_review_replies
-- Description: Add reply and like functionality to vendor reviews

ALTER TABLE vendor_reviews 
ADD COLUMN IF NOT EXISTS reply_text TEXT,
ADD COLUMN IF NOT EXISTS replied_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS helpful_count INTEGER DEFAULT 0;

-- Add index for better filtering
CREATE INDEX IF NOT EXISTS idx_vendor_reviews_is_public ON vendor_reviews(is_public);
