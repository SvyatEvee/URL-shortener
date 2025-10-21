CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    refresh_token_randnom_part_hash  UNIQUE NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_user_id       ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_token ON sessions(refresh_token_randnom_part_hash);
CREATE INDEX IF NOT EXISTS idx_expires_at    ON sessions(expires_at);

