-- Migration: 019_internal_notes
-- Description: Add internal_notes column to quote_requests table for private vendor notes.

ALTER TABLE "quote_requests" ADD COLUMN IF NOT EXISTS "internal_notes" TEXT;
