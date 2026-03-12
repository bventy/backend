-- Create system_incidents table to log outages automatically
CREATE TABLE IF NOT EXISTS "public"."system_incidents" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "monitor_name" text NOT NULL,
    "issue_type" text NOT NULL, -- e.g. "Down", "Performance Degraded"
    "description" text,
    "status" text DEFAULT 'investigating' NOT NULL, -- investigating, identified, monitoring, resolved
    "created_at" timestamp DEFAULT now(),
    "resolved_at" timestamp,
    CONSTRAINT "system_incidents_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

-- Index for finding active incidents quickly
CREATE INDEX idx_active_incidents ON public.system_incidents (monitor_name) WHERE resolved_at IS NULL;
