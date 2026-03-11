-- 024_performance_indices.sql
-- Optimizing common marketplace query patterns

-- 1. Events: Filtering by organizer and status
CREATE INDEX IF NOT EXISTS idx_events_organizer_status ON events (organizer_user_id, status);
CREATE INDEX IF NOT EXISTS idx_events_date ON events (event_date);

-- 2. Quote Requests: High-frequency joins and status filters
CREATE INDEX IF NOT EXISTS idx_quote_requests_vendor_status ON quote_requests (vendor_id, status);
CREATE INDEX IF NOT EXISTS idx_quote_requests_organizer_status ON quote_requests (organizer_user_id, status);
CREATE INDEX IF NOT EXISTS idx_quote_requests_event_id ON quote_requests (event_id);

-- 3. Vendor Profiles: Slug-based lookups and category filtering
CREATE INDEX IF NOT EXISTS idx_vendor_profiles_category ON vendor_profiles (category);
CREATE INDEX IF NOT EXISTS idx_vendor_profiles_city ON vendor_profiles (city);

-- 4. Messaging: Sorting by last_message_at for inbox view
-- (Existing: idx_conversations_last_message - ensuring it covers quote_id as well for joins)
CREATE INDEX IF NOT EXISTS idx_conversations_quote_id ON conversations (quote_id);

-- 5. Activity Log: Performance metrics lookups
CREATE INDEX IF NOT EXISTS idx_platform_activity_entity_created ON platform_activity_log (entity_type, entity_id, created_at);
