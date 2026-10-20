DROP INDEX IF EXISTS idx_pulse_users_email;
ALTER TABLE pulse_users DROP COLUMN IF EXISTS password_hash;
