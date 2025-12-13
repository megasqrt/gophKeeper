-- Откат миграции: удаляем checksum
ALTER TABLE login_passwords
    DROP COLUMN IF EXISTS checksum;

