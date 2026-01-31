CREATE TABLE IF NOT EXISTS url (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    alias TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    CONSTRAINT uq_alias_user UNIQUE (alias, user_id)
);

CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
CREATE INDEX IF NOT EXISTS idx_user_id ON url(user_id)

