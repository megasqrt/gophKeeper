-- Удаляем колонку version из таблицы binary_data
ALTER TABLE binary_data DROP COLUMN IF EXISTS version;
