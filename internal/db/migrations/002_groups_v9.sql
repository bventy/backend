-- V9 Group Management Updates

-- 1. Add updated_at to group_members
ALTER TABLE "group_members" ADD COLUMN IF NOT EXISTS "updated_at" timestamp DEFAULT now();

-- 2. Group Invites Table
CREATE TABLE "public"."group_invites" (
    "id" uuid DEFAULT uuid_generate_v4() NOT NULL,
    "group_id" uuid NOT NULL,
    "invited_email" text NOT NULL,
    "role" text NOT NULL CHECK (role IN ('member', 'manager')),
    "status" text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired')),
    "invited_by" uuid NOT NULL,
    "created_at" timestamp DEFAULT now(),
    CONSTRAINT "group_invites_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "group_invites_group_id_fkey" FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT "group_invites_invited_by_fkey" FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE CASCADE
) WITH (oids = false);

CREATE INDEX idx_group_invites_email ON public.group_invites USING btree (invited_email);
CREATE INDEX idx_group_invites_group ON public.group_invites USING btree (group_id);
