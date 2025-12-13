-- Удаляем старое поле data из таблицы text_data, так как теперь используются title и text
ALTER TABLE text_data
    DROP COLUMN IF EXISTS data;

