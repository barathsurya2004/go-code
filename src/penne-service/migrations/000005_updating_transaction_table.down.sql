DROP INDEX IF EXISTS idx_transactionrows_envelope_id;
DROP INDEX IF EXISTS idx_transactionrows_user_id;

ALTER TABLE transactionrows DROP COLUMN IF EXISTS envelope_id;
ALTER TABLE transactionrows ADD COLUMN category varchar(255) NOT NULL DEFAULT 'uncategorized';
ALTER TABLE transactionrows RENAME COLUMN user_id TO user_uuid;
ALTER TABLE transactionrows RENAME COLUMN id TO uuid;
