-- Откат миграции: удаляем добавленные поля
ALTER TABLE binary_data
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS size,
    DROP COLUMN IF EXISTS checksum;

