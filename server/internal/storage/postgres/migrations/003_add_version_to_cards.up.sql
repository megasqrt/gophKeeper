-- Добавляем колонку version в таблицу bank_cards
ALTER TABLE bank_cards ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1 NOT NULL;
