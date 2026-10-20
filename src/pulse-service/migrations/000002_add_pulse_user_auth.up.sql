ALTER TABLE pulse_users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pulse_users_email ON pulse_users(email) WHERE email IS NOT NULL AND email != '';
