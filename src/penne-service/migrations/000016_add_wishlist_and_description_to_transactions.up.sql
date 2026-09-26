ALTER TABLE transactionrows
    ADD COLUMN IF NOT EXISTS description VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS wishlist_item_id UUID REFERENCES wishlist_items(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transactionrows_wishlist_item_id ON transactionrows(wishlist_item_id);

ALTER TABLE wishlist_allocations
    ADD COLUMN IF NOT EXISTS transaction_id UUID REFERENCES transactionrows(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_wishlist_allocations_transaction_id ON wishlist_allocations(transaction_id);
