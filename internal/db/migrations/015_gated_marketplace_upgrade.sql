-- Gated Marketplace Upgrade Migration

-- 1. Create platform_activity_log if missing
CREATE TABLE IF NOT EXISTS platform_activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    entity_type VARCHAR(50) NOT NULL, -- 'vendor', 'quote', 'event', etc.
    entity_id UUID NOT NULL,
    action_type VARCHAR(50) NOT NULL, -- 'view', 'quote_created', 'quote_responded', 'quote_accepted', 'contact_unlocked'
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_platform_activity_log_entity ON platform_activity_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_platform_activity_log_action ON platform_activity_log(action_type);
CREATE INDEX IF NOT EXISTS idx_platform_activity_log_actor ON platform_activity_log(actor_user_id);

-- 2. Update quote_requests with missing columns
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='quote_requests' AND column_name='special_requirements') THEN
        ALTER TABLE quote_requests ADD COLUMN special_requirements TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='quote_requests' AND column_name='deadline') THEN
        ALTER TABLE quote_requests ADD COLUMN deadline TIMESTAMP WITH TIME ZONE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='quote_requests' AND column_name='attachment_url') THEN
        ALTER TABLE quote_requests ADD COLUMN attachment_url TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='quote_requests' AND column_name='revision_requested_at') THEN
        ALTER TABLE quote_requests ADD COLUMN revision_requested_at TIMESTAMP WITH TIME ZONE;
    END IF;
END $$;

-- 3. Ensure events has completed_at (checked: it does per inspection, but safe to guard)
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='events' AND column_name='completed_at') THEN
        ALTER TABLE events ADD COLUMN completed_at TIMESTAMP WITH TIME ZONE;
    END IF;
END $$;
