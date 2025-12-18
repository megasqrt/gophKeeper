-- Удаляем колонку version из таблицы text_data
ALTER TABLE text_data DROP COLUMN IF EXISTS version;
