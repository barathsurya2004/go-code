DROP INDEX IF EXISTS idx_wishlist_allocations_transaction_id;
ALTER TABLE wishlist_allocations DROP COLUMN IF EXISTS transaction_id;

DROP INDEX IF EXISTS idx_transactionrows_wishlist_item_id;
ALTER TABLE transactionrows DROP COLUMN IF EXISTS wishlist_item_id;
ALTER TABLE transactionrows DROP COLUMN IF EXISTS description;
