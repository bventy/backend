-- 12. Vendor Gallery Images
CREATE TABLE "public"."vendor_gallery_images" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "vendor_id" uuid NOT NULL,
    "image_url" text NOT NULL,
    "caption" text,
    "sort_order" int DEFAULT 0,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_gallery_images_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendor_gallery_images_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE
) WITH (oids = false);

-- 13. Vendor Portfolio Files (PDFs)
CREATE TABLE "public"."vendor_portfolio_files" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "vendor_id" uuid NOT NULL,
    "file_url" text NOT NULL,
    "title" text NOT NULL,
    "sort_order" int DEFAULT 0,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "vendor_portfolio_files_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "vendor_portfolio_files_vendor_id_fkey" FOREIGN KEY (vendor_id) REFERENCES vendor_profiles(id) ON DELETE CASCADE
) WITH (oids = false);

-- Resize profile_image_url if needed? No, text is fine.
