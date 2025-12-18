-- Удаляем колонку version из таблицы login_passwords
ALTER TABLE login_passwords DROP COLUMN IF EXISTS version;
