-- Migration: 016_email_logs
-- Description: Create a table to log all sent emails with a 30-day retention context.

CREATE TABLE IF NOT EXISTS "email_logs" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "to_email" TEXT NOT NULL,
    "subject" TEXT NOT NULL,
    "body_html" TEXT NOT NULL,
    "template_key" VARCHAR(50),
    "sent_at" TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_email_logs_sent_at ON email_logs(sent_at);
CREATE INDEX IF NOT EXISTS idx_email_logs_to_email ON email_logs(to_email);

-- Optional: Initial cleanup of any orphaned data if needed, 
-- but since this is a new table, we just ensure it's ready.
