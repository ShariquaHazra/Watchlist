ALTER TABLE user_sessions
ADD COLUMN IF NOT EXISTS jti VARCHAR(64);

UPDATE user_sessions
SET jti = session_id
WHERE jti IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_sessions_jti
ON user_sessions(jti);