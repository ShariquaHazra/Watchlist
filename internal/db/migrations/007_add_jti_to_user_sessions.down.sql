DROP INDEX IF EXISTS idx_user_sessions_jti;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS jti;