CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    envelope_id UUID REFERENCES envelope(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    amount_e5 BIGINT NOT NULL,
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',
    next_billing_date DATE NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'bank_card',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    auto_renew BOOLEAN NOT NULL DEFAULT true,
    notes TEXT,
    last_charged_at TIMESTAMPTZ,
    last_transaction_id UUID REFERENCES transactionrows(id) ON DELETE SET NULL,
    merchant_pattern VARCHAR(120),
    charge_window_hours INT NOT NULL DEFAULT 48,
    occurrence_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_uuid ON subscriptions(user_uuid);
CREATE INDEX IF NOT EXISTS idx_subscriptions_next_billing ON subscriptions(next_billing_date);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
