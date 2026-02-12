-- Adminer 5.4.2 PostgreSQL 16.11 dump

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Drop everything first (Cascade)
DROP TABLE IF EXISTS "event_shortlisted_vendors" CASCADE;
DROP TABLE IF EXISTS "events" CASCADE;
DROP TABLE IF EXISTS "group_members" CASCADE;
DROP TABLE IF EXISTS "groups" CASCADE;
DROP TABLE IF EXISTS "vendor_portfolio_images" CASCADE; -- Keeping if needed, though not in V8 spec, good to have cleanup
DROP TABLE IF EXISTS "organizer_bookmarks" CASCADE; -- Cleanup old
DROP TABLE IF EXISTS "organizer_profiles" CASCADE; -- Cleanup old
DROP TABLE IF EXISTS "vendor_profiles" CASCADE;
DROP TABLE IF EXISTS "user_permissions" CASCADE;
DROP TABLE IF EXISTS "permissions" CASCADE;
DROP TABLE IF EXISTS "users" CASCADE;

-- 1. Users Table
CREATE TABLE "public"."users" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "email" text NOT NULL,
    "password_hash" text NOT NULL,
    "full_name" text NOT NULL,
    "username" text,
    "phone" text,
    "city" text,
    "bio" text,
    "profile_image_url" text,
    "role" text NOT NULL DEFAULT 'user',
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "users_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "users_email_key" UNIQUE ("email"),
    CONSTRAINT "users_username_key" UNIQUE ("username"),
    CONSTRAINT "users_role_check" CHECK (role IN ('user', 'staff', 'admin', 'super_admin'))
) WITH (oids = false);

-- 2. Permissions & User Permissions (Kept for Admin Verification requirement)
CREATE TABLE "public"."permissions" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "code" text NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "permissions_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "permissions_code_key" UNIQUE ("code")
) WITH (oids = false);

CREATE TABLE "public"."user_permissions" (
    "user_id" uuid NOT NULL,
    "permission_id" uuid NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "user_permissions_pkey" PRIMARY KEY ("user_id", "permission_id"),
    CONSTRAINT "user_permissions_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT "user_permissions_permission_id_fkey" FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) WITH (oids = false);

-- 3. Vendor Profiles
CREATE TABLE "public"."vendor_profiles" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "owner_user_id" uuid NOT NULL,
    "business_name" text NOT NULL,
    "slug" text NOT NULL,
    "category" text NOT NULL,
    "city" text NOT NULL,
    "whatsapp_link" text NOT NULL,
    "bio" text,
    "status" text DEFAULT 'pending' NOT NULL,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_profiles_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendor_profiles_owner_user_id_key" UNIQUE ("owner_user_id"),
    CONSTRAINT "vendor_profiles_slug_key" UNIQUE ("slug"),
    CONSTRAINT "vendor_profiles_owner_user_id_fkey" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT "vendor_profiles_status_check" CHECK (status IN ('pending', 'verified', 'rejected'))
) WITH (oids = false);

CREATE INDEX idx_vendor_status ON public.vendor_profiles USING btree (status);
CREATE INDEX idx_vendor_slug ON public.vendor_profiles USING btree (slug);

-- 4. Groups (Communities)
CREATE TABLE "public"."groups" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "name" text NOT NULL,
    "slug" text NOT NULL,
    "description" text,
    "city" text,
    "owner_user_id" uuid NOT NULL,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "groups_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "groups_slug_key" UNIQUE ("slug"),
    CONSTRAINT "groups_owner_user_id_fkey" FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
) WITH (oids = false);

-- 5. Group Members
CREATE TABLE "public"."group_members" (
    "group_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "role" text NOT NULL DEFAULT 'member',
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "group_members_pkey" PRIMARY KEY ("group_id", "user_id"),
    CONSTRAINT "group_members_group_id_fkey" FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT "group_members_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT "group_members_role_check" CHECK (role IN ('member', 'manager', 'owner'))
) WITH (oids = false);

-- 6. Events
CREATE TABLE "public"."events" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "title" text NOT NULL,
    "city" text NOT NULL,
    "date" timestamp NOT NULL,
    "budget" numeric, -- Simplified from min/max to generic budget field or we can add min/max if preferred. Prompt says "budget". Wait, prompt says "budget" in TABLES list, but "budget_min, budget_max" in REQUEST. I will use budget_min and budget_max to be safe + generic budget text/numeric. Let's use min/max numeric.
    "budget_min" numeric,
    "budget_max" numeric,
    "event_type" text,
    "organizer_user_id" uuid,
    "organizer_group_id" uuid,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "events_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "events_organizer_user_id_fkey" FOREIGN KEY (organizer_user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT "events_organizer_group_id_fkey" FOREIGN KEY (organizer_group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT "events_owner_check" CHECK (
        (organizer_user_id IS NOT NULL AND organizer_group_id IS NULL) OR
        (organizer_user_id IS NULL AND organizer_group_id IS NOT NULL)
    )
) WITH (oids = false);

-- 7. Event Shortlisted Vendors
CREATE TABLE "public"."event_shortlisted_vendors" (
    "event_id" uuid NOT NULL,
    "vendor_id" uuid NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "event_shortlisted_vendors_pkey" PRIMARY KEY ("event_id", "vendor_id"),
    CONSTRAINT "event_shortlisted_vendors_event_id_fkey" FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    CONSTRAINT "event_shortlisted_vendors_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE
) WITH (oids = false);


-- Initial Data
INSERT INTO permissions (code) VALUES
('vendor.verify')
ON CONFLICT (code) DO NOTHING;
