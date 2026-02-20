-- Database Reset and Seed Script (Pro Version - Expanded & Clean)
-- Password for all users: 123pass
-- Bcrypt hash: $2a$10$UcyPfjnlNkgM3/bE3P59JO7LALQr1k0h77r5wQl.LCwGe8eEQtDAO

BEGIN;

-- TRUNCATE all tables to start fresh
TRUNCATE TABLE 
    users, 
    vendor_profiles, 
    groups, 
    events, 
    group_invites, 
    group_members, 
    permissions, 
    user_permissions, 
    vendor_gallery_images, 
    vendor_portfolio_files, 
    vendor_portfolio_images, 
    event_shortlisted_vendors 
CASCADE;

-- Enable UUID extension if not present
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. SEED USERS
DO $$
DECLARE
    pass_hash TEXT := '$2a$10$UcyPfjnlNkgM3/bE3P59JO7LALQr1k0h77r5wQl.LCwGe8eEQtDAO';
    uid_super_admin UUID := uuid_generate_v4();
    uid_admin UUID := uuid_generate_v4();
    uid_vendor1 UUID := uuid_generate_v4();
    uid_vendor2 UUID := uuid_generate_v4();
    uid_vendor3 UUID := uuid_generate_v4();
    uid_vendor4 UUID := uuid_generate_v4();
    uid_vendor5 UUID := uuid_generate_v4();
    uid_vendor6 UUID := uuid_generate_v4();
    uid_vendor7 UUID := uuid_generate_v4();
    uid_vendor8 UUID := uuid_generate_v4();
    uid_vendor9 UUID := uuid_generate_v4();
    uid_vendor10 UUID := uuid_generate_v4();
    uid_pending1 UUID := uuid_generate_v4();
    uid_pending2 UUID := uuid_generate_v4();
    uid_pending3 UUID := uuid_generate_v4();
    uid_user1 UUID := uuid_generate_v4();
    uid_user2 UUID := uuid_generate_v4();
    uid_user3 UUID := uuid_generate_v4();
BEGIN
    INSERT INTO users (id, email, password_hash, full_name, username, role, city) VALUES
    (uid_super_admin, 'superadmin@gmail.com', pass_hash, 'Bventy Super Admin', 'superadmin', 'super_admin', 'Pune'),
    (uid_admin, 'admin@gmail.com', pass_hash, 'Pune City Admin', 'admin_pune', 'admin', 'Pune'),
    (uid_vendor1, 'vendor1@gmail.com', pass_hash, 'Arun Patil', 'snapshot_pune', 'user', 'Pune'),
    (uid_vendor2, 'vendor2@gmail.com', pass_hash, 'Sanjay Deshmukh', 'dreamdecor', 'user', 'Pune'),
    (uid_vendor3, 'vendor3@gmail.com', pass_hash, 'Meera Joshi', 'grandpavilion', 'user', 'Pune'),
    (uid_vendor4, 'vendor4@gmail.com', pass_hash, 'Kunal Shah', 'puneplate', 'user', 'Pune'),
    (uid_vendor5, 'vendor5@gmail.com', pass_hash, 'Rahul Verma', 'beats_pune', 'user', 'Pune'),
    (uid_vendor6, 'vendor6@gmail.com', pass_hash, 'Priya Kulkarni', 'makeup_art', 'user', 'Pune'),
    (uid_vendor7, 'vendor7@gmail.com', pass_hash, 'Amit Gadgil', 'lighting_pro', 'user', 'Pune'),
    (uid_vendor8, 'vendor8@gmail.com', pass_hash, 'Sneha Tare', 'floral_vibes', 'user', 'Pune'),
    (uid_vendor9, 'vendor9@gmail.com', pass_hash, 'Vikram Malhotra', 'transport_pro', 'user', 'Pune'),
    (uid_vendor10, 'vendor10@gmail.com', pass_hash, 'Aditi Rao', 'invitation_studio', 'user', 'Pune'),
    (uid_pending1, 'pending1@gmail.com', pass_hash, 'Suresh Kadam', 'creative_cakes', 'user', 'Pune'),
    (uid_pending2, 'pending2@gmail.com', pass_hash, 'Anita Shinde', 'sparkle_lights', 'user', 'Pune'),
    (uid_pending3, 'pending3@gmail.com', pass_hash, 'Rohan Mehta', 'royal_banquets', 'user', 'Pune'),
    (uid_user1, 'user1@gmail.com', pass_hash, 'Rahul Kulkarni', 'rahul_k', 'user', 'Pune'),
    (uid_user2, 'user2@gmail.com', pass_hash, 'Anjali More', 'anjali_m', 'user', 'Pune'),
    (uid_user3, 'user3@gmail.com', pass_hash, 'Vikram Singh', 'vikram_s', 'user', 'Pune');

    -- 2. SEED VENDOR PROFILES (No portfolio_image_url to allow fallback)
    INSERT INTO vendor_profiles (owner_user_id, business_name, slug, category, city, bio, whatsapp_link, status) VALUES
    (uid_vendor1, 'SnapShot Studio', 'snapshot-studio-pune', 'Photography', 'Pune', 'Premium wedding and event photography. Capturing moments that last a lifetime.', 'https://wa.me/919800000001', 'verified'),
    (uid_vendor2, 'Dream Decorators', 'dream-decorators-pune', 'Decor', 'Pune', 'Specialists in floral and thematic event decoration for weddings and corporate events.', 'https://wa.me/919800000002', 'verified'),
    (uid_vendor3, 'The Grand Pavillion', 'the-grand-pavillion-pune', 'Venue', 'Pune', 'A premium banquet hall in Baner, perfect for weddings, birthdays, and seminars.', 'https://wa.me/919800000003', 'verified'),
    (uid_vendor4, 'Pune Plate Catering', 'pune-plate-catering-pune', 'Catering', 'Pune', 'Authentic multi-cuisine catering with a focus on hygiene and taste.', 'https://wa.me/919800000004', 'verified'),
    (uid_vendor5, 'Pune Beats DJ', 'pune-beats-dj', 'DJ', 'Pune', 'Professional sound and lighting for parties, weddings, and corporate gigs.', 'https://wa.me/919800000005', 'verified'),
    (uid_vendor6, 'Glow & Glam Makeup', 'glow-glam-makeup', 'Makeup', 'Pune', 'Bridal and party makeup by certified professionals with 10+ years of experience.', 'https://wa.me/919800000006', 'verified'),
    (uid_vendor7, 'Stellar Lights', 'stellar-lights-pune', 'Decor', 'Pune', 'Ambient and stage lighting experts for large scale outdoor events.', 'https://wa.me/919800000007', 'verified'),
    (uid_vendor8, 'Petals & Props', 'petals-props-pune', 'Decor', 'Pune', 'Luxurious floral arrangements and prop rentals for boutique events.', 'https://wa.me/919800000008', 'verified'),
    (uid_vendor9, 'SafeRide Transport', 'saferide-transport', 'Logistics', 'Pune', 'Reliable guest transport and luxury car rentals for events in Pune.', 'https://wa.me/919800000009', 'verified'),
    (uid_vendor10, 'Ink & Paper Invite Studio', 'ink-paper-invite', 'Stationery', 'Pune', 'Custom wedding invitations and event stationery with a modern touch.', 'https://wa.me/919800000010', 'verified'),
    (uid_pending1, 'Creative Cakes', 'creative-cakes-pune', 'Catering', 'Pune', 'Delicious custom cakes and desserts for all occasions.', 'https://wa.me/919800000011', 'pending'),
    (uid_pending2, 'Sparkle Lights', 'sparkle-lights-pune', 'Decor', 'Pune', 'Making your events shine with our unique lighting solutions.', 'https://wa.me/919800000012', 'pending'),
    (uid_pending3, 'Royal Banquets', 'royal-banquets-pune', 'Venue', 'Pune', 'A grand setting for your unforgettable celebrations.', 'https://wa.me/919800000013', 'pending');

    -- 3. SEED GROUPS
    INSERT INTO groups (name, slug, description, city, owner_user_id) VALUES
    ('Pune Event Planners', 'pune-event-planners', 'A community of event organizers and vendors in Pune.', 'Pune', uid_user1),
    ('Baner Wedding Hub', 'baner-wedding-hub', 'Connecting couples with the best vendors in Baner and Balewadi.', 'Pune', uid_user2);

    -- 4. SEED EVENTS (Expanded for user1)
    INSERT INTO events (title, event_type, city, event_date, budget_min, budget_max, status, organizer_user_id) VALUES
    ('Anya & Sameer Wedding', 'Wedding', 'Pune', '2026-12-15', 500000, 1500000, 'planning', uid_user1),
    ('Tech Meetup 2026', 'Corporate', 'Pune', '2026-08-10', 50000, 150000, 'planning', uid_user1),
    ('Annual Sports Day', 'Sports', 'Pune', '2027-01-20', 100000, 300000, 'draft', uid_user1),
    ('Grand Charity Auction', 'Gala', 'Pune', '2026-10-30', 200000, 500000, 'planning', uid_user1),
    ('Pune Startup Awards', 'Corporate', 'Pune', '2026-09-12', 300000, 800000, 'planning', uid_user1),
    ('Summer Music Fest', 'Festival', 'Pune', '2026-05-20', 1000000, 2500000, 'draft', uid_user1),
    ('Corporate Gala 2026', 'Corporate', 'Pune', '2026-11-20', 200000, 500000, 'planning', uid_user2),
    ('Sangeet Night', 'Wedding', 'Pune', '2026-12-14', 100000, 400000, 'planning', uid_user2);

END $$;

COMMIT;
