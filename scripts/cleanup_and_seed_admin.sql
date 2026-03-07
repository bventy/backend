-- 1. Purge user-generated data but keep templates and settings
-- Use TRUNCATE with CASCADE to handle foreign key dependencies

TRUNCATE TABLE 
    conversations,
    email_logs,
    email_otps,
    event_shortlisted_vendors,
    event_views,
    events,
    group_invites,
    group_members,
    groups,
    internal_notes,
    message_reactions,
    message_reads,
    messages,
    messaging_conversations,
    messaging_messages,
    platform_activity_log,
    quote_requests,
    user_permissions,
    vendor_calendar_blocks,
    vendor_cancellation_policies,
    vendor_contact_clicks,
    vendor_gallery_images,
    vendor_oauth_connections,
    vendor_portfolio_files,
    vendor_pricing_rules,
    vendor_profile_views,
    vendor_profiles,
    vendor_reviews,
    vendor_service_areas,
    vendor_services,
    users
CASCADE;

-- 2. Insert new super_admin
INSERT INTO users (email, password_hash, role, email_verified, full_name, created_at, updated_at)
VALUES (
    'admin@bventy.in', 
    '$2a$10$3m3bBN3/er24caz7HkzzDuJN9cCfR80nBtyU8XEW7UimD8sKcu5Fy', 
    'super_admin', 
    true, 
    'Bventy Admin', 
    NOW(), 
    NOW()
);

-- 3. Insert corresponding vendor account
DO $$
DECLARE
    admin_id UUID;
BEGIN
    SELECT id INTO admin_id FROM users WHERE email = 'admin@bventy.in';

    INSERT INTO vendor_profiles (
        owner_user_id, 
        business_name, 
        slug, 
        category,
        city,
        whatsapp_link,
        status, 
        created_at, 
        updated_at
    ) VALUES (
        admin_id, 
        'Bventy Event Management', 
        'bventy-events', 
        'Platform Administration',
        'Pune',
        'https://wa.me/910000000000',
        'verified', 
        NOW(), 
        NOW()
    );
END $$;
