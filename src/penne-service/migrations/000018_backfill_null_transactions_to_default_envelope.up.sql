-- Backfill any transactionrows with NULL envelope_id to the user's system "default" envelope
UPDATE transactionrows t
SET envelope_id = e.id
FROM envelope e
WHERE t.envelope_id IS NULL
  AND e.user_uuid = t.user_id
  AND e.is_system = true
  AND lower(e.name) = 'default';

-- Recalculate spent_amount_e5 on system default allocations
UPDATE allocation a
SET spent_amount_e5 = COALESCE((
    SELECT SUM(t.amount_e5)
    FROM transactionrows t
    WHERE t.envelope_id = a.envelope_id
      AND t.txn_type = 'debit'
      AND t.created_at::date >= a.start_date
      AND t.created_at::date <= a.end_date
), 0),
updated_at = NOW()
WHERE a.envelope_id IN (
    SELECT id FROM envelope WHERE is_system = true AND lower(name) = 'default'
);
