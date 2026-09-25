-- 1. For users with existing credit transactions and monthly_budget_e5 = 0, backfill from credits
UPDATE users u
SET monthly_budget_e5 = sub.credit_sum,
    salary_day = 1
FROM (
    SELECT user_id, SUM(amount_e5)::bigint as credit_sum
    FROM transactionrows
    WHERE txn_type = 'credit'
    GROUP BY user_id
) sub
WHERE u.uuid = sub.user_id AND u.monthly_budget_e5 = 0;

-- 2. For users without credits but having envelope targets, backfill from envelope targets sum
UPDATE users u
SET monthly_budget_e5 = sub.target_sum::bigint,
    salary_day = 1
FROM (
    SELECT user_uuid, SUM(target_amount_e5)::bigint as target_sum
    FROM envelope
    GROUP BY user_uuid
) sub
WHERE u.uuid = sub.user_uuid AND u.monthly_budget_e5 = 0 AND sub.target_sum > 0;
