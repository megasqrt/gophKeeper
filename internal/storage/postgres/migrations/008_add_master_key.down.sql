-- Откат: удаляем поле
ALTER TABLE users
    DROP COLUMN IF EXISTS encrypted_master_key;

