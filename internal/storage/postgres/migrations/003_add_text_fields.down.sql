-- Откат миграции: удаляем добавленные поля
ALTER TABLE text_data
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS text,
    DROP COLUMN IF EXISTS checksum;

