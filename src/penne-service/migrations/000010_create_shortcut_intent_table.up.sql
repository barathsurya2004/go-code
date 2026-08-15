CREATE TABLE IF NOT EXISTS shortcut_intent (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    envelope_id UUID REFERENCES envelope(id) ON DELETE SET NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    status VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    transaction_id UUID REFERENCES transactionrows(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_shortcut_intent_user_id ON shortcut_intent(user_id);
CREATE INDEX IF NOT EXISTS idx_shortcut_intent_envelope_id ON shortcut_intent(envelope_id);
CREATE INDEX IF NOT EXISTS idx_shortcut_intent_transaction_id ON shortcut_intent(transaction_id);

ALTER TABLE transactionrows ADD COLUMN IF NOT EXISTS shortcut_intent_id UUID REFERENCES shortcut_intent(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transactionrows_shortcut_intent_id ON transactionrows(shortcut_intent_id);
