-- Add telemetry columns to system_status_history for better diagnostics
ALTER TABLE "public"."system_status_history" 
ADD COLUMN IF NOT EXISTS "latency_ms" integer,
ADD COLUMN IF NOT EXISTS "error_msg" text;
