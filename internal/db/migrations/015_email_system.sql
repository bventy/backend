-- Migration: 015_email_system
-- Description: Add email verification fields, OTP tracking, and template management.

-- 1. Update users table
ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "email_verified" BOOLEAN DEFAULT FALSE;
ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "email_verified_at" TIMESTAMP WITH TIME ZONE;
ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "email_verification_attempts" INT DEFAULT 0;

-- 2. Create email_otps table
CREATE TABLE IF NOT EXISTS "email_otps" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id" UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "email" TEXT NOT NULL,
    "code" VARCHAR(6) NOT NULL,
    "purpose" TEXT NOT NULL CHECK (purpose IN ('verify', 'reset')),
    "expires_at" TIMESTAMP WITH TIME ZONE NOT NULL,
    "attempts" INT DEFAULT 0,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_email_otps_user_id ON email_otps(user_id);
CREATE INDEX IF NOT EXISTS idx_email_otps_email ON email_otps(email);

-- 3. Create email_templates table
CREATE TABLE IF NOT EXISTS "email_templates" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "template_key" VARCHAR(50) UNIQUE NOT NULL,
    "subject" TEXT NOT NULL,
    "body_html" TEXT NOT NULL,
    "is_enabled" BOOLEAN DEFAULT TRUE,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT now(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Seed initial templates
INSERT INTO email_templates (template_key, subject, body_html) VALUES
('verify_email', 'Verify your Bventy account', '<html><body><h1>Welcome to Bventy!</h1><p>Your verification code is: <strong>{{code}}</strong></p><p>This code expires in 10 minutes.</p></body></html>'),
('reset_password', 'Password Reset Request', '<html><body><h1>Password Reset</h1><p>Reset your password using the code: <strong>{{code}}</strong></p><p>If you did not request this, please ignore this email.</p></body></html>'),
('quote_requested', 'New Quote Request Received', '<html><body><h1>New Quote Request</h1><p>Hello {{vendor_name}}, you have received a new quote request for "{{event_title}}".</p><p>Check your dashboard for details.</p></body></html>'),
('quote_updated', 'Quote Updated', '<html><body><h1>Quote Updated</h1><p>The quote for "{{event_title}}" has been updated.</p></body></html>'),
('quote_accepted', 'Quote Accepted!', '<html><body><h1>Congratulations!</h1><p>Your quote for "{{event_title}}" has been accepted by {{organizer_name}}.</p></body></html>'),
('quote_rejected', 'Quote Update', '<html><body><p>Your quote for "{{event_title}}" was not selected at this time.</p></body></html>')
ON CONFLICT (template_key) DO NOTHING;

-- 4. Create platform_settings table
CREATE TABLE IF NOT EXISTS "platform_settings" (
    "key" VARCHAR(100) PRIMARY KEY,
    "value" TEXT NOT NULL,
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Seed platform settings
INSERT INTO platform_settings (key, value) VALUES
('quote_email_notifications_enabled', 'true')
ON CONFLICT (key) DO NOTHING;
