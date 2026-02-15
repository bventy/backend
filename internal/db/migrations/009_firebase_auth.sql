-- Add firebase_uid column and remove password_hash
ALTER TABLE users ADD COLUMN firebase_uid VARCHAR(255) UNIQUE;
ALTER TABLE users DROP COLUMN password_hash;
