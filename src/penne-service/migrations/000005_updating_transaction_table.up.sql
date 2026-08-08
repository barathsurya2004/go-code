ALTER TABLE transactionrows RENAME COLUMN uuid TO id;
ALTER TABLE transactionrows RENAME COLUMN user_uuid TO user_id;
ALTER TABLE transactionrows DROP COLUMN IF EXISTS category;
ALTER TABLE transactionrows ADD COLUMN envelope_id uuid REFERENCES envelope(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transactionrows_envelope_id ON transactionrows(envelope_id);
CREATE INDEX IF NOT EXISTS idx_transactionrows_user_id ON transactionrows(user_id);
