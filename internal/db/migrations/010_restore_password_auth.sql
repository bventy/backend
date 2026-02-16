-- Restore password_hash column if missing (reverting Firebase auth schema change)
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
