-- Migration: 014_schema_fixes
-- Description: Add missing columns and tables to ensure schema consistency with handlers

-- 1. Events Table Updates
ALTER TABLE "events" ADD COLUMN IF NOT EXISTS "status" TEXT DEFAULT 'upcoming' CHECK (status IN ('upcoming', 'completed', 'cancelled'));
ALTER TABLE "events" ADD COLUMN IF NOT EXISTS "completed_at" TIMESTAMP WITH TIME ZONE;

-- 2. Create quote_requests if missing (re-ensuring standard structure)
CREATE TABLE IF NOT EXISTS "quote_requests" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "event_id" UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    "vendor_id" UUID NOT NULL REFERENCES vendor_profiles(id) ON DELETE CASCADE,
    "organizer_user_id" UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "message" TEXT NOT NULL,
    "quoted_price" NUMERIC,
    "vendor_response" TEXT,
    "status" TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'responded', 'accepted', 'rejected', 'revision_requested', 'archived')),
    "responded_at" TIMESTAMP WITH TIME ZONE,
    "accepted_at" TIMESTAMP WITH TIME ZONE,
    "rejected_at" TIMESTAMP WITH TIME ZONE,
    "revision_requested_at" TIMESTAMP WITH TIME ZONE,
    "revision_message" TEXT,
    "contact_unlocked_at" TIMESTAMP WITH TIME ZONE,
    "contact_expires_at" TIMESTAMP WITH TIME ZONE,
    "archived_at" TIMESTAMP WITH TIME ZONE,
    "budget_range" TEXT,
    "special_requirements" TEXT,
    "deadline" TEXT,
    "attachment_url" TEXT,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Create platform_activity_log if missing
CREATE TABLE IF NOT EXISTS "platform_activity_log" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "entity_type" TEXT NOT NULL,
    "entity_id" UUID,
    "action_type" TEXT NOT NULL,
    "actor_user_id" UUID REFERENCES users(id) ON DELETE SET NULL,
    "metadata" JSONB,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_quote_requests_org ON quote_requests(organizer_user_id);
CREATE INDEX IF NOT EXISTS idx_quote_requests_vend ON quote_requests(vendor_id);
CREATE INDEX IF NOT EXISTS idx_activity_actor ON platform_activity_log(actor_user_id);
