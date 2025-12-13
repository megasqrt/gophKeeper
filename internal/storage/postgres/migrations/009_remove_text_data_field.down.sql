-- Откат: добавляем обратно поле data (но это не рекомендуется, так как данные будут потеряны)
ALTER TABLE text_data
    ADD COLUMN IF NOT EXISTS data TEXT;

