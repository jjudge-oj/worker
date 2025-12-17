ALTER TABLE users
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS role;

DROP INDEX IF EXISTS idx_users_role;
