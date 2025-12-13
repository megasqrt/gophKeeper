-- Добавляем поле для хранения зашифрованного мастер-ключа пользователя
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS encrypted_master_key BYTEA;

