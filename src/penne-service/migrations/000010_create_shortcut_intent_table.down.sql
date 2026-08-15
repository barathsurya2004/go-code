DROP INDEX IF EXISTS idx_transactionrows_shortcut_intent_id;

ALTER TABLE transactionrows DROP COLUMN IF EXISTS shortcut_intent_id;

DROP INDEX IF EXISTS idx_shortcut_intent_transaction_id;
DROP INDEX IF EXISTS idx_shortcut_intent_envelope_id;
DROP INDEX IF EXISTS idx_shortcut_intent_user_id;

DROP TABLE IF EXISTS shortcut_intent;
