-- Database Restoration Script: Defaults & Initial Seed
-- This script restores lost configuration data (email templates, platform settings)
-- and optionally re-seeds initial users and vendors.

BEGIN;

-- 1. Restore Email Templates
INSERT INTO email_templates (template_key, subject, body_html, from_name, from_email) VALUES
('verify_email', 'Verify your Bventy account', '<html><body><h1>Welcome to Bventy!</h1><p>Your verification code is: <strong>{{code}}</strong></p><p>This code expires in 60 minutes.</p></body></html>', 'Bventy Security', 'security@bventy.in'),
('reset_password', 'Password Reset Request', '<html><body><h1>Password Reset</h1><p>Reset your password using the code: <strong>{{code}}</strong></p><p>This code expires in 60 minutes. If you did not request this, please ignore this email.</p></body></html>', 'Bventy Security', 'security@bventy.in'),
('quote_requested', 'New Quote Request Received', '<html><body><h1>New Quote Request</h1><p>Hello {{vendor_name}}, you have received a new quote request for "{{event_title}}".</p><p>Check your dashboard for details.</p></body></html>', 'Bventy Notifications', 'notifications@bventy.in'),
('quote_updated', 'Quote Updated', '<html><body><h1>Quote Updated</h1><p>The quote for "{{event_title}}" has been updated.</p></body></html>', 'Bventy Notifications', 'notifications@bventy.in'),
('quote_accepted', 'Quote Accepted!', '<html><body><h1>Congratulations!</h1><p>Your quote for "{{event_title}}" has been accepted by {{organizer_name}}.</p></body></html>', 'Bventy Notifications', 'notifications@bventy.in'),
('quote_rejected', 'Quote Update', '<html><body><p>Your quote for "{{event_title}}" was not selected at this time.</p></body></html>', 'Bventy Notifications', 'notifications@bventy.in')
ON CONFLICT (template_key) DO UPDATE SET
    subject = EXCLUDED.subject,
    body_html = EXCLUDED.body_html,
    from_name = EXCLUDED.from_name,
    from_email = EXCLUDED.from_email,
    updated_at = NOW();

-- 2. Restore Platform Settings
INSERT INTO platform_settings (key, value) VALUES
('quote_email_notifications_enabled', 'true')
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = NOW();

-- 3. (Optional) Re-seed initial Users and Vendors from seed.sql logic
-- Enabling UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$
DECLARE
    pass_hash TEXT := '$2a$10$UcyPfjnlNkgM3/bE3P59JO7LALQr1k0h77r5wQl.LCwGe8eEQtDAO';
    uid_super_admin UUID := uuid_generate_v4();
    uid_admin UUID := uuid_generate_v4();
    uid_vendor1 UUID := uuid_generate_v4();
    uid_user1 UUID := uuid_generate_v4();
BEGIN
    -- Only insert if they don't exist
    IF NOT EXISTS (SELECT 1 FROM users WHERE email = 'superadmin@gmail.com') THEN
        INSERT INTO users (id, email, password_hash, full_name, username, role, city, email_verified) VALUES
        (uid_super_admin, 'superadmin@gmail.com', pass_hash, 'Bventy Super Admin', 'superadmin', 'super_admin', 'Pune', true);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM users WHERE email = 'admin@gmail.com') THEN
        INSERT INTO users (id, email, password_hash, full_name, username, role, city, email_verified) VALUES
        (uid_admin, 'admin@gmail.com', pass_hash, 'Pune City Admin', 'admin_pune', 'admin', 'Pune', true);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM users WHERE email = 'vendor1@gmail.com') THEN
        INSERT INTO users (id, email, password_hash, full_name, username, role, city, email_verified) VALUES
        (uid_vendor1, 'vendor1@gmail.com', pass_hash, 'Arun Patil', 'snapshot_pune', 'user', 'Pune', true);

        INSERT INTO vendor_profiles (owner_user_id, business_name, slug, category, city, bio, whatsapp_link, status) VALUES
        (uid_vendor1, 'SnapShot Studio', 'snapshot-studio-pune', 'Photography', 'Pune', 'Premium wedding and event photography. Capturing moments that last a lifetime.', 'https://wa.me/919800000001', 'verified');
    END IF;
END $$;

COMMIT;
