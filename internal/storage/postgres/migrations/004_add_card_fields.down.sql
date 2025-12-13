-- Откат миграции: удаляем checksum
ALTER TABLE bank_cards
    DROP COLUMN IF EXISTS checksum;

