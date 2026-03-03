CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE public.conversations (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    quote_id uuid NOT NULL UNIQUE,
    vendor_id uuid NOT NULL,
    organizer_user_id uuid,
    organizer_group_id uuid,
    chat_locked boolean DEFAULT true,
    created_at timestamptz DEFAULT now(),
    last_message_at timestamptz,
    CONSTRAINT conversations_quote_fkey
        FOREIGN KEY (quote_id) REFERENCES quote_requests(id)
        ON DELETE CASCADE
);

CREATE TABLE public.messages (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id uuid NOT NULL,
    sender_user_id uuid,
    message_type text NOT NULL DEFAULT 'text',
    body text,
    attachment_url text,
    attachment_type text,
    system_payload jsonb,
    created_at timestamptz DEFAULT now(),
    edited_at timestamptz,
    deleted_at timestamptz,
    CONSTRAINT messages_conversation_fkey FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    CONSTRAINT messages_sender_fkey FOREIGN KEY (sender_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE public.message_reads (
    message_id uuid NOT NULL,
    user_id uuid NOT NULL,
    read_at timestamptz DEFAULT now(),
    PRIMARY KEY (message_id, user_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_conversations_last_message ON conversations(last_message_at DESC);
