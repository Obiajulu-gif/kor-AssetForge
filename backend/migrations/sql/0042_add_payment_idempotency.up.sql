-- Payment idempotency keys (#339)
-- Prevents duplicate charges when a create-payment request is retried:
-- every payment must carry a unique (user_id, idempotency_key) pair.
ALTER TABLE payments ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

-- Backfill rows created before idempotency support so the NOT NULL and
-- unique constraints can be applied without failing.
UPDATE payments
SET idempotency_key = 'legacy_' || id::text
WHERE idempotency_key IS NULL;

ALTER TABLE payments ALTER COLUMN idempotency_key SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_user_idempotency_key
    ON payments (user_id, idempotency_key);
