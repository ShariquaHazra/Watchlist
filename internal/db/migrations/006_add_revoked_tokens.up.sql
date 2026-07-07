CREATE TABLE IF NOT EXISTS revoked_tokens (
    id         SERIAL PRIMARY KEY,
    jti        VARCHAR(64) NOT NULL UNIQUE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    revoked_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_jti ON revoked_tokens(jti);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);