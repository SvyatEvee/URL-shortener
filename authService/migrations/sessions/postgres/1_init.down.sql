-- Удаляем индексы для таблицы sessions
DROP INDEX IF EXISTS idx_expires_at;
DROP INDEX IF EXISTS idx_refresh_token;
DROP INDEX IF EXISTS idx_user_id;

-- Удаляем таблицу sessions
DROP TABLE IF EXISTS sessions;
