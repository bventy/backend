-- Migration 021: Refine Vendor Cancellation Policies
ALTER TABLE "public"."vendor_cancellation_policies" 
ADD COLUMN IF NOT EXISTS "strictness_level" text DEFAULT 'flexible',
ADD COLUMN IF NOT EXISTS "time_frame_days" integer DEFAULT 1,
ADD COLUMN IF NOT EXISTS "refund_percentage" integer DEFAULT 100;

-- Update existing records to match the current 'flexible' default
UPDATE "public"."vendor_cancellation_policies" SET 
    strictness_level = 'flexible',
    time_frame_days = 1,
    refund_percentage = 100 
WHERE strictness_level IS NULL;
