-- Persistent tracking for system maintenance tasks
CREATE TABLE IF NOT EXISTS system_maintenance (
    key TEXT PRIMARY KEY,
    last_run_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Initialize tracking keys for backups
INSERT INTO system_maintenance (key) VALUES ('last_full_backup'), ('last_schema_backup')
ON CONFLICT (key) DO NOTHING;
