-- Adminer 5.4.2 PostgreSQL 16.11 dump

DROP FUNCTION IF EXISTS "uuid_generate_v1";
CREATE FUNCTION "uuid_generate_v1" () RETURNS uuid LANGUAGE c AS 'uuid_generate_v1';

DROP FUNCTION IF EXISTS "uuid_generate_v1mc";
CREATE FUNCTION "uuid_generate_v1mc" () RETURNS uuid LANGUAGE c AS 'uuid_generate_v1mc';

DROP FUNCTION IF EXISTS "uuid_generate_v3";
CREATE FUNCTION "uuid_generate_v3" (IN "namespace" uuid, IN "name" text) RETURNS uuid LANGUAGE c AS 'uuid_generate_v3';

DROP FUNCTION IF EXISTS "uuid_generate_v4";
CREATE FUNCTION "uuid_generate_v4" () RETURNS uuid LANGUAGE c AS 'uuid_generate_v4';

DROP FUNCTION IF EXISTS "uuid_generate_v5";
CREATE FUNCTION "uuid_generate_v5" (IN "namespace" uuid, IN "name" text) RETURNS uuid LANGUAGE c AS 'uuid_generate_v5';

DROP FUNCTION IF EXISTS "uuid_nil";
CREATE FUNCTION "uuid_nil" () RETURNS uuid LANGUAGE c AS 'uuid_nil';

DROP FUNCTION IF EXISTS "uuid_ns_dns";
CREATE FUNCTION "uuid_ns_dns" () RETURNS uuid LANGUAGE c AS 'uuid_ns_dns';

DROP FUNCTION IF EXISTS "uuid_ns_oid";
CREATE FUNCTION "uuid_ns_oid" () RETURNS uuid LANGUAGE c AS 'uuid_ns_oid';

DROP FUNCTION IF EXISTS "uuid_ns_url";
CREATE FUNCTION "uuid_ns_url" () RETURNS uuid LANGUAGE c AS 'uuid_ns_url';

DROP FUNCTION IF EXISTS "uuid_ns_x500";
CREATE FUNCTION "uuid_ns_x500" () RETURNS uuid LANGUAGE c AS 'uuid_ns_x500';

DROP TABLE IF EXISTS "organizer_bookmarks";
CREATE TABLE "public"."organizer_bookmarks" (
    "organizer_id" uuid NOT NULL,
    "vendor_id" uuid NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "organizer_bookmarks_pkey" PRIMARY KEY ("organizer_id", "vendor_id")
)
WITH (oids = false);


DROP TABLE IF EXISTS "organizers";
CREATE TABLE "public"."organizers" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "user_id" uuid NOT NULL,
    "name" text NOT NULL,
    "organization" text,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "organizers_pkey" PRIMARY KEY ("id")
)
WITH (oids = false);

CREATE UNIQUE INDEX organizers_user_id_key ON public.organizers USING btree (user_id);


DROP TABLE IF EXISTS "users";
CREATE TABLE "public"."users" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "email" text NOT NULL,
    "password_hash" text NOT NULL,
    "role" text NOT NULL,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "users_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "users_role_check" CHECK (((role = ANY (ARRAY['organizer'::text, 'vendor'::text, 'admin'::text]))))
)
WITH (oids = false);

CREATE UNIQUE INDEX users_email_key ON public.users USING btree (email);


DROP TABLE IF EXISTS "vendor_portfolio_images";
CREATE TABLE "public"."vendor_portfolio_images" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "vendor_id" uuid NOT NULL,
    "image_url" text NOT NULL,
    "position" integer DEFAULT '0',
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_portfolio_images_pkey" PRIMARY KEY ("id")
)
WITH (oids = false);


DROP TABLE IF EXISTS "vendors";
CREATE TABLE "public"."vendors" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "user_id" uuid NOT NULL,
    "name" text NOT NULL,
    "category" text NOT NULL,
    "city" text NOT NULL,
    "bio" text,
    "whatsapp_link" text NOT NULL,
    "status" text DEFAULT 'pending' NOT NULL,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "vendors_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendors_status_check" CHECK (((status = ANY (ARRAY['pending'::text, 'verified'::text, 'rejected'::text]))))
)
WITH (oids = false);

CREATE UNIQUE INDEX vendors_user_id_key ON public.vendors USING btree (user_id);

CREATE INDEX idx_vendor_city ON public.vendors USING btree (city);

CREATE INDEX idx_vendor_category ON public.vendors USING btree (category);

CREATE INDEX idx_vendor_status ON public.vendors USING btree (status);


ALTER TABLE ONLY "public"."organizer_bookmarks" ADD CONSTRAINT "organizer_bookmarks_organizer_id_fkey" FOREIGN KEY (organizer_id) REFERENCES organizers(id) ON DELETE CASCADE;
ALTER TABLE ONLY "public"."organizer_bookmarks" ADD CONSTRAINT "organizer_bookmarks_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE CASCADE;

ALTER TABLE ONLY "public"."organizers" ADD CONSTRAINT "organizers_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY "public"."vendor_portfolio_images" ADD CONSTRAINT "vendor_portfolio_images_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE CASCADE;

ALTER TABLE ONLY "public"."vendors" ADD CONSTRAINT "vendors_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
