CREATE TABLE IF NOT EXISTS wishlist_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    target_amount_e5 BIGINT NOT NULL,
    saved_amount_e5 BIGINT NOT NULL DEFAULT 0,
    priority INT NOT NULL DEFAULT 3 CHECK (priority BETWEEN 1 AND 5),
    urgency INT NOT NULL DEFAULT 3 CHECK (urgency BETWEEN 1 AND 5),
    item_type VARCHAR(50) NOT NULL DEFAULT 'purchase',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    target_date DATE,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wishlist_items_user_uuid ON wishlist_items(user_uuid);
CREATE INDEX IF NOT EXISTS idx_wishlist_items_status ON wishlist_items(status);

CREATE TABLE IF NOT EXISTS wishlist_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wishlist_item_id UUID NOT NULL REFERENCES wishlist_items(id) ON DELETE CASCADE,
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    amount_e5 BIGINT NOT NULL,
    source_type VARCHAR(50) NOT NULL DEFAULT 'cycle_surplus',
    cycle_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wishlist_allocations_item_id ON wishlist_allocations(wishlist_item_id);
CREATE INDEX IF NOT EXISTS idx_wishlist_allocations_user_uuid ON wishlist_allocations(user_uuid);
