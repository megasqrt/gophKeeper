-- Добавляем колонку version в таблицу login_passwords
ALTER TABLE login_passwords ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1 NOT NULL;
