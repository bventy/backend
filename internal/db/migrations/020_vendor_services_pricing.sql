-- 20. Vendor Services & Pricing

-- Services Table
CREATE TABLE IF NOT EXISTS "public"."vendor_services" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "vendor_id" uuid NOT NULL,
    "name" text NOT NULL,
    "base_price" numeric NOT NULL,
    "price_unit" text NOT NULL DEFAULT '/ Plate',
    "status" text NOT NULL DEFAULT 'active',
    "description" text,
    "created_at" timestamp DEFAULT now(),
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_services_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendor_services_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE,
    CONSTRAINT "vendor_services_status_check" CHECK (status IN ('active', 'draft', 'inactive'))
) WITH (oids = false);

-- Pricing Rules Table
CREATE TABLE IF NOT EXISTS "public"."vendor_pricing_rules" (
    "vendor_id" uuid NOT NULL,
    "weekend_premium_enabled" boolean DEFAULT false,
    "weekend_premium_percentage" numeric DEFAULT 15,
    "last_minute_booking_enabled" boolean DEFAULT false,
    "last_minute_booking_percentage" numeric DEFAULT 20,
    "last_minute_days" int DEFAULT 7,
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_pricing_rules_pkey" PRIMARY KEY ("vendor_id"),
    CONSTRAINT "vendor_pricing_rules_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE
) WITH (oids = false);

-- Cancellation Policies Table
CREATE TABLE IF NOT EXISTS "public"."vendor_cancellation_policies" (
    "vendor_id" uuid NOT NULL,
    "policy_type" text NOT NULL DEFAULT 'moderate',
    "custom_text" text,
    "updated_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_cancellation_policies_pkey" PRIMARY KEY ("vendor_id"),
    CONSTRAINT "vendor_cancellation_policies_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE,
    CONSTRAINT "vendor_cancellation_policies_type_check" CHECK (policy_type IN ('flexible', 'moderate', 'strict', 'custom'))
) WITH (oids = false);

-- Service Areas Table
CREATE TABLE IF NOT EXISTS "public"."vendor_service_areas" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "vendor_id" uuid NOT NULL,
    "area_name" text NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_service_areas_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendor_service_areas_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE
) WITH (oids = false);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_vendor_services_vendor_id ON public.vendor_services USING btree (vendor_id);
CREATE INDEX IF NOT EXISTS idx_vendor_service_areas_vendor_id ON public.vendor_service_areas USING btree (vendor_id);
