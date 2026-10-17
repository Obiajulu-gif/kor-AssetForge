-- Migration 0041: Fix unique constraints on users for soft delete support (#310)
-- Replaces unconditional unique constraints with partial unique indexes (WHERE deleted_at IS NULL),
-- enabling users to recreate accounts with the same email, username, or Stellar address after deletion.

-- Drop existing unconditional unique constraints
ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_users_email;
ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_users_username;
ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_users_stellar_address;

-- Drop any unconditional unique indexes that may have been created
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_stellar_address;
DROP INDEX IF EXISTS uq_users_email;
DROP INDEX IF EXISTS uq_users_username;
DROP INDEX IF EXISTS uq_users_stellar_address;

-- Create partial unique indexes that only apply to active (non-deleted) accounts
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_stellar_address ON users (stellar_address) WHERE deleted_at IS NULL;
