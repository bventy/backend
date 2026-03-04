-- Add booking availability and view count to vendor profiles
ALTER TABLE vendor_profiles ADD COLUMN is_accepting_bookings BOOLEAN DEFAULT TRUE;
ALTER TABLE vendor_profiles ADD COLUMN views_count BIGINT DEFAULT 0;

-- Index for faster sorted listings
CREATE INDEX idx_vendor_profiles_availability ON vendor_profiles(is_accepting_bookings DESC);
