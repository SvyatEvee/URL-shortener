-- Удаляем индексы для таблицы sessions
DROP INDEX IF EXISTS idx_expires_at;
DROP INDEX IF EXISTS idx_refresh_token;
DROP INDEX IF EXISTS idx_user_id;

-- Удаляем таблицу sessions
DROP TABLE IF EXISTS sessions;

-- Удаляем индекс для таблицы users
DROP INDEX IF EXISTS idx_email;

-- Удаляем таблицу users
DROP TABLE IF EXISTS users;

-- Удаляем таблицу roles
DROP TABLE IF EXISTS roles;