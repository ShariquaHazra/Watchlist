-- Har user ka ek session per device_type (mobile/desktop)
-- UNIQUE(user_id, device_type) ki wajah se ek user ek device_type par
-- sirf 1 active session rakh sakta hai. Naya login hote hi purani row
-- REPLACE (upsert) ho jaati hai, isi se purana session automatically invalid ho jata hai.

CREATE TABLE IF NOT EXISTS user_sessions (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_type  VARCHAR(10) NOT NULL CHECK (device_type IN ('mobile', 'desktop')),
    session_id   VARCHAR(64) NOT NULL,
    device_info  TEXT,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, device_type)
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);