-- Удаляем колонку version из таблицы bank_cards
ALTER TABLE bank_cards DROP COLUMN IF EXISTS version;
