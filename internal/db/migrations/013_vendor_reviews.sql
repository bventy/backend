-- Migration: 013_vendor_reviews
-- Description: Add table for vendor reviews and ratings

CREATE TABLE IF NOT EXISTS vendor_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID NOT NULL REFERENCES vendor_profiles(id) ON DELETE CASCADE,
    organizer_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quote_id UUID REFERENCES quote_requests(id) ON DELETE SET NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_vendor_reviews_vendor_id ON vendor_reviews(vendor_id);
CREATE INDEX IF NOT EXISTS idx_vendor_reviews_organizer_user_id ON vendor_reviews(organizer_user_id);

-- Constraint: One review per user per vendor? 
-- Let's stick to allowing multiple if they have multiple events, but for now a simple unique constraint per user/vendor is cleaner for ratings.
-- Actually, let's allow multiple as someone might hire a vendor for multiple events.
