-- Add media columns to vendor_profiles for Dashboard support

ALTER TABLE vendor_profiles 
ADD COLUMN IF NOT EXISTS portfolio_image_url TEXT,
ADD COLUMN IF NOT EXISTS gallery_images TEXT[] DEFAULT '{}',
ADD COLUMN IF NOT EXISTS portfolio_files JSONB DEFAULT '[]';

-- Ensure defaults are applied to existing rows
UPDATE vendor_profiles SET gallery_images = '{}' WHERE gallery_images IS NULL;
UPDATE vendor_profiles SET portfolio_files = '[]'::jsonb WHERE portfolio_files IS NULL;
