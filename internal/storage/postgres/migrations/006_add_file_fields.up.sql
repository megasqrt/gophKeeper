-- Добавляем поля name, size, checksum в таблицу binary_data
ALTER TABLE binary_data
    ADD COLUMN IF NOT EXISTS name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS size BIGINT,
    ADD COLUMN IF NOT EXISTS checksum VARCHAR(64);

