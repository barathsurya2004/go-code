-- Standalone Idempotent Backfill Script for Older Accounts
-- Can be executed safely on local or production database.

BEGIN;

-- 1. Fix zero timestamps on envelope and allocation tables
UPDATE envelope
SET created_at = COALESCE((SELECT u.created_at FROM users u WHERE u.uuid = envelope.user_uuid), NOW())
WHERE created_at < '1970-01-01';

UPDATE envelope
SET updated_at = created_at
WHERE updated_at < '1970-01-01';

UPDATE allocation
SET created_at = COALESCE((SELECT e.created_at FROM envelope e WHERE e.id = allocation.envelope_id), NOW())
WHERE created_at < '1970-01-01';

UPDATE allocation
SET updated_at = created_at
WHERE updated_at < '1970-01-01';

-- 2. Standardize existing system envelope groups and envelopes
UPDATE envelope
SET is_system = true, name = 'default', cadence = 'forever'
WHERE is_system = false AND name = '' AND target_amount_e5 = 0;

UPDATE envelope
SET name = 'default'
WHERE is_system = true AND (name IS NULL OR name = '' OR name = 'Unallocated');

UPDATE envelope
SET cadence = 'forever'
WHERE is_system = true AND cadence != 'forever';

UPDATE envelope_group eg
SET is_system = true
WHERE EXISTS (
    SELECT 1 FROM envelope e
    WHERE e.envelope_group_id = eg.id AND e.is_system = true
) AND eg.is_system = false;

-- 3. Populate missing system envelope groups, envelopes, allocations, and tokens for older users
INSERT INTO envelope_group (id, user_uuid, name, is_system, created_at, updated_at)
SELECT gen_random_uuid(), u.uuid, 'default', true, u.created_at, u.created_at
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM envelope_group eg
    WHERE eg.user_uuid = u.uuid AND eg.is_system = true
);

INSERT INTO envelope (id, user_uuid, envelope_group_id, name, is_system, target_amount_e5, cadence, country_iso, created_at, updated_at)
SELECT
    gen_random_uuid(),
    u.uuid,
    (SELECT eg.id FROM envelope_group eg WHERE eg.user_uuid = u.uuid AND eg.is_system = true ORDER BY eg.created_at LIMIT 1),
    'default',
    true,
    0,
    'forever',
    'IN',
    u.created_at,
    u.created_at
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM envelope e
    WHERE e.user_uuid = u.uuid AND e.is_system = true
);

INSERT INTO allocation (id, envelope_id, allocated_amount_e5, spent_amount_e5, start_date, end_date, created_at, updated_at)
SELECT
    gen_random_uuid(),
    e.id,
    0,
    0,
    e.created_at::date,
    (e.created_at + interval '100 years')::date,
    e.created_at,
    e.created_at
FROM envelope e
WHERE e.is_system = true
  AND NOT EXISTS (
      SELECT 1 FROM allocation a WHERE a.envelope_id = e.id
  );

INSERT INTO user_tokens (user_id, token_uuid, prefix, name, scopes, created_at, updated_at)
SELECT
    u.uuid,
    gen_random_uuid(),
    'penne_at',
    'default',
    '{"all"}',
    u.created_at,
    u.created_at
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM user_tokens ut WHERE ut.user_id = u.uuid
);

-- 4. Align start_date and truncate end_date for 100-year allocations on non-system envelopes
UPDATE allocation a
SET 
    start_date = CASE
        WHEN e.cadence = 'monthly' THEN date_trunc('month', a.start_date)::date
        WHEN e.cadence = 'weekly'  THEN (a.start_date - EXTRACT(DOW FROM a.start_date)::integer)::date
        WHEN e.cadence = 'yearly'  THEN date_trunc('year', a.start_date)::date
        WHEN e.cadence = 'daily'   THEN a.start_date
        ELSE a.start_date
    END,
    end_date = CASE
        WHEN e.cadence = 'monthly' THEN (date_trunc('month', a.start_date) + interval '1 month - 1 day')::date
        WHEN e.cadence = 'weekly'  THEN (a.start_date - EXTRACT(DOW FROM a.start_date)::integer + 6)::date
        WHEN e.cadence = 'yearly'  THEN (date_trunc('year', a.start_date) + interval '1 year - 1 day')::date
        WHEN e.cadence = 'daily'   THEN a.start_date
        ELSE (date_trunc('month', a.start_date) + interval '1 month - 1 day')::date
    END,
    updated_at = NOW()
FROM envelope e
WHERE a.envelope_id = e.id
  AND e.is_system = false
  AND (a.end_date - a.start_date) > 366;

-- Also align start_date to cadence cycle boundary for non-system allocations if misaligned
UPDATE allocation a
SET 
    start_date = CASE
        WHEN e.cadence = 'monthly' THEN date_trunc('month', a.start_date)::date
        WHEN e.cadence = 'weekly'  THEN (a.start_date - EXTRACT(DOW FROM a.start_date)::integer)::date
        WHEN e.cadence = 'yearly'  THEN date_trunc('year', a.start_date)::date
        ELSE a.start_date
    END,
    updated_at = NOW()
FROM envelope e
WHERE a.envelope_id = e.id
  AND e.is_system = false
  AND (
      (e.cadence = 'monthly' AND a.start_date != date_trunc('month', a.start_date)::date) OR
      (e.cadence = 'weekly' AND a.start_date != (a.start_date - EXTRACT(DOW FROM a.start_date)::integer)::date) OR
      (e.cadence = 'yearly' AND a.start_date != date_trunc('year', a.start_date)::date)
  );

-- 5. Generate missing monthly allocations up to CURRENT_DATE for monthly non-system envelopes
INSERT INTO allocation (id, envelope_id, allocated_amount_e5, spent_amount_e5, start_date, end_date, created_at, updated_at)
SELECT
    gen_random_uuid(),
    e.id,
    e.target_amount_e5,
    0,
    m.cycle_start::date,
    (m.cycle_start + interval '1 month - 1 day')::date,
    m.cycle_start::timestamptz,
    NOW()
FROM envelope e
CROSS JOIN LATERAL (
    SELECT generate_series(
        date_trunc('month', COALESCE((SELECT MIN(a.start_date) FROM allocation a WHERE a.envelope_id = e.id), e.created_at::date)),
        date_trunc('month', CURRENT_DATE),
        interval '1 month'
    ) AS cycle_start
) m
WHERE e.is_system = false
  AND e.cadence = 'monthly'
  AND NOT EXISTS (
      SELECT 1 FROM allocation existing
      WHERE existing.envelope_id = e.id
        AND daterange(existing.start_date, existing.end_date, '[]') &&
            daterange(m.cycle_start::date, (m.cycle_start + interval '1 month - 1 day')::date, '[]')
  );

-- 6. Generate missing weekly allocations up to CURRENT_DATE for weekly non-system envelopes
INSERT INTO allocation (id, envelope_id, allocated_amount_e5, spent_amount_e5, start_date, end_date, created_at, updated_at)
SELECT
    gen_random_uuid(),
    e.id,
    e.target_amount_e5,
    0,
    w.cycle_start::date,
    (w.cycle_start + interval '6 days')::date,
    w.cycle_start::timestamptz,
    NOW()
FROM envelope e
CROSS JOIN LATERAL (
    SELECT generate_series(
        (COALESCE((SELECT MIN(a.start_date) FROM allocation a WHERE a.envelope_id = e.id), e.created_at::date) - EXTRACT(DOW FROM COALESCE((SELECT MIN(a.start_date) FROM allocation a WHERE a.envelope_id = e.id), e.created_at::date))::integer)::date,
        (CURRENT_DATE - EXTRACT(DOW FROM CURRENT_DATE)::integer)::date,
        interval '1 week'
    ) AS cycle_start
) w
WHERE e.is_system = false
  AND e.cadence = 'weekly'
  AND NOT EXISTS (
      SELECT 1 FROM allocation existing
      WHERE existing.envelope_id = e.id
        AND daterange(existing.start_date, existing.end_date, '[]') &&
            daterange(w.cycle_start::date, (w.cycle_start + interval '6 days')::date, '[]')
  );

-- 7. Recalculate spent_amount_e5 across ALL allocations from transactionrows
UPDATE allocation a
SET spent_amount_e5 = COALESCE((
    SELECT SUM(t.amount_e5)
    FROM transactionrows t
    WHERE t.envelope_id = a.envelope_id
      AND t.txn_type = 'debit'
      AND t.created_at::date >= a.start_date
      AND t.created_at::date <= a.end_date
), 0),
updated_at = NOW();

COMMIT;
