-- Migration: 024_fix_events_schema
-- Description: Rename 'date' to 'event_date' and add 'cover_image_url' for consistency with handlers.

DO $$ 
BEGIN 
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='events' AND column_name='date') THEN
        ALTER TABLE "events" RENAME COLUMN "date" TO "event_date";
    END IF;
END $$;

ALTER TABLE "events" ADD COLUMN IF NOT EXISTS "cover_image_url" TEXT;
