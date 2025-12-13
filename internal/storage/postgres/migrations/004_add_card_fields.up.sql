-- Добавляем поле checksum в таблицу bank_cards
ALTER TABLE bank_cards
    ADD COLUMN IF NOT EXISTS checksum VARCHAR(64);

