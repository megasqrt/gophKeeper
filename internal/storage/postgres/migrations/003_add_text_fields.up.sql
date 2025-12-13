-- Добавляем поля title, text, checksum в таблицу text_data
ALTER TABLE text_data
    ADD COLUMN IF NOT EXISTS title VARCHAR(255),
    ADD COLUMN IF NOT EXISTS text TEXT,
    ADD COLUMN IF NOT EXISTS checksum VARCHAR(64);

-- Если в data уже есть данные, можно попробовать их мигрировать
-- Но для новых записей будем использовать новые поля

