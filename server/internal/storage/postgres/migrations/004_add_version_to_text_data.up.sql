-- Добавляем колонку version в таблицу text_data
ALTER TABLE text_data ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1 NOT NULL;
