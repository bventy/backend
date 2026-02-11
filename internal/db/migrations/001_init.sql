-- Adminer 5.4.2 PostgreSQL 16.11 dump

-- Extensions (Ignore errors if they exist)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Users Table
DROP TABLE IF EXISTS "users" CASCADE;
CREATE TABLE "public"."users" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "email" text NOT NULL,
    "password_hash" text NOT NULL,
    "role" text NOT NULL DEFAULT 'user',
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "users_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "users_email_key" UNIQUE ("email"),
    CONSTRAINT "users_role_check" CHECK (role IN ('user', 'staff', 'admin', 'super_admin'))
) WITH (oids = false);

-- 2. Permissions Table
DROP TABLE IF EXISTS "permissions" CASCADE;
CREATE TABLE "public"."permissions" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "code" text NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "permissions_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "permissions_code_key" UNIQUE ("code")
) WITH (oids = false);

-- 3. User Permissions (Many-to-Many)
DROP TABLE IF EXISTS "user_permissions" CASCADE;
CREATE TABLE "public"."user_permissions" (
    "user_id" uuid NOT NULL,
    "permission_id" uuid NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "user_permissions_pkey" PRIMARY KEY ("user_id", "permission_id"),
    CONSTRAINT "user_permissions_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT "user_permissions_permission_id_fkey" FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) WITH (oids = false);

-- 4. Vendor Profiles
DROP TABLE IF EXISTS "vendor_profiles" CASCADE;
CREATE TABLE "public"."vendor_profiles" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "user_id" uuid NOT NULL,
    "name" text NOT NULL,
    "slug" text NOT NULL,
    "category" text NOT NULL,
    "city" text NOT NULL,
    "bio" text,
    "whatsapp_link" text NOT NULL,
    "status" text DEFAULT 'pending' NOT NULL,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_profiles_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendor_profiles_user_id_key" UNIQUE ("user_id"),
    CONSTRAINT "vendor_profiles_slug_key" UNIQUE ("slug"),
    CONSTRAINT "vendor_profiles_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT "vendor_profiles_status_check" CHECK (status IN ('pending', 'verified', 'rejected'))
) WITH (oids = false);

CREATE INDEX idx_vendor_status ON public.vendor_profiles USING btree (status);
CREATE INDEX idx_vendor_slug ON public.vendor_profiles USING btree (slug);

-- 5. Organizer Profiles
DROP TABLE IF EXISTS "organizer_profiles" CASCADE;
CREATE TABLE "public"."organizer_profiles" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "user_id" uuid NOT NULL,
    "display_name" text NOT NULL,
    "city" text NOT NULL,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "organizer_profiles_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "organizer_profiles_user_id_key" UNIQUE ("user_id"),
    CONSTRAINT "organizer_profiles_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) WITH (oids = false);

-- 6. Organizer Bookmarks
DROP TABLE IF EXISTS "organizer_bookmarks" CASCADE;
CREATE TABLE "public"."organizer_bookmarks" (
    "organizer_id" uuid NOT NULL,
    "vendor_id" uuid NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "organizer_bookmarks_pkey" PRIMARY KEY ("organizer_id", "vendor_id"),
    CONSTRAINT "organizer_bookmarks_organizer_id_fkey" FOREIGN KEY (organizer_id) REFERENCES organizer_profiles(id) ON DELETE CASCADE,
    CONSTRAINT "organizer_bookmarks_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE
) WITH (oids = false);

-- 7. Vendor Portfolio Images
DROP TABLE IF EXISTS "vendor_portfolio_images" CASCADE;
CREATE TABLE "public"."vendor_portfolio_images" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "vendor_id" uuid NOT NULL,
    "image_url" text NOT NULL,
    "position" integer DEFAULT 0,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_portfolio_images_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendor_portfolio_images_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE
) WITH (oids = false);

-- Initial Permissions Data
INSERT INTO permissions (code) VALUES
('vendor.onboard.review'),
('vendor.verify'),
('vendor.manage'),
('staff.manage'),
('admin.manage')
ON CONFLICT (code) DO NOTHING;
