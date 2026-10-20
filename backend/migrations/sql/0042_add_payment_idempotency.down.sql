-- Revert payment idempotency keys (#339)
DROP INDEX IF EXISTS idx_payments_user_idempotency_key;
ALTER TABLE payments DROP COLUMN IF EXISTS idempotency_key;
