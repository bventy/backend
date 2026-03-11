-- Create system_status_history table to store periodic health checks
CREATE TABLE IF NOT EXISTS "public"."system_status_history" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "monitor_name" text NOT NULL,
    "status" text NOT NULL,
    "checked_at" timestamp DEFAULT now(),
    CONSTRAINT "system_status_history_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

-- Index for faster lookups by monitor name and timestamp
CREATE INDEX idx_status_history_monitor_time ON public.system_status_history USING btree (monitor_name, checked_at DESC);
