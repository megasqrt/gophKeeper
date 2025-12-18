-- Добавляем колонку version в таблицу binary_data
ALTER TABLE binary_data ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1 NOT NULL;
