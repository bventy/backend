-- Up Migration
CREATE TABLE IF NOT EXISTS quote_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    vendor_id UUID NOT NULL REFERENCES vendor_profiles(id) ON DELETE CASCADE,
    organizer_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    quoted_price NUMERIC(12, 2),
    vendor_response TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, responded, accepted, rejected
    responded_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quote_requests_event_id ON quote_requests(event_id);
CREATE INDEX IF NOT EXISTS idx_quote_requests_vendor_id ON quote_requests(vendor_id);
CREATE INDEX IF NOT EXISTS idx_quote_requests_organizer_id ON quote_requests(organizer_user_id);

CREATE TABLE IF NOT EXISTS platform_activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL, -- 'quote', 'vendor', 'event', etc.
    entity_id UUID NOT NULL,
    action_type VARCHAR(50) NOT NULL, -- 'quote_created', 'view', 'contact_click', 'shortlist', etc.
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- optional, if logged in
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_platform_activity_log_entity_type_id ON platform_activity_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_platform_activity_log_action_type ON platform_activity_log(action_type);
CREATE INDEX IF NOT EXISTS idx_platform_activity_log_actor_id ON platform_activity_log(actor_user_id);
