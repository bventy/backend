-- Migration: Add per-template from_name and from_email
ALTER TABLE email_templates ADD COLUMN IF NOT EXISTS from_name TEXT;
ALTER TABLE email_templates ADD COLUMN IF NOT EXISTS from_email TEXT;

-- Set default values for existing templates
UPDATE email_templates SET from_name = 'Bventy Security', from_email = 'security@bventy.in' WHERE template_key IN ('verify_email', 'reset_password');
UPDATE email_templates SET from_name = 'Bventy Notifications', from_email = 'notifications@bventy.in' WHERE template_key NOT IN ('verify_email', 'reset_password');
