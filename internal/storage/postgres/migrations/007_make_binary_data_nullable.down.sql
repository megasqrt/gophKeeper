-- Откат: возвращаем NOT NULL
ALTER TABLE binary_data
    ALTER COLUMN data SET NOT NULL;

