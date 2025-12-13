-- Делаем поле data в binary_data nullable, так как мы храним только метаданные на сервере
ALTER TABLE binary_data
    ALTER COLUMN data DROP NOT NULL;

