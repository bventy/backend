-- Migration: 024_fix_events_schema
-- Description: Rename 'date' to 'event_date' and add 'cover_image_url' for consistency with handlers.

ALTER TABLE "events" RENAME COLUMN "date" TO "event_date";
ALTER TABLE "events" ADD COLUMN IF NOT EXISTS "cover_image_url" TEXT;
