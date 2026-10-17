-- Rollback 0041: Revert partial unique indexes back to full unique constraints (#310)

DROP INDEX IF EXISTS idx_users_stellar_address;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;

-- Restore unconditional unique constraints
ALTER TABLE users ADD CONSTRAINT uq_users_email UNIQUE (email);
ALTER TABLE users ADD CONSTRAINT uq_users_username UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT uq_users_stellar_address UNIQUE (stellar_address);
