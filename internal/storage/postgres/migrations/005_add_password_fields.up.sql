-- Добавляем поле checksum в таблицу login_passwords
ALTER TABLE login_passwords
    ADD COLUMN IF NOT EXISTS checksum VARCHAR(64);

