-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    encrypted_master_key BYTEA,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

-- Создание таблицы для хранения пар логин/пароль
CREATE TABLE IF NOT EXISTS login_passwords (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    login_data TEXT NOT NULL,
    password_data TEXT NOT NULL,
    metadata TEXT,
    checksum VARCHAR(64),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- Создание таблицы для истории изменений login_passwords
-- CREATE TABLE IF NOT EXISTS login_passwords_history (
--     history_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     id UUID NOT NULL, -- ID оригинальной записи
--     user_id UUID NOT NULL,
--     login_data TEXT NOT NULL,
--     password_data TEXT NOT NULL,
--     metadata TEXT,
--     created_at BIGINT NOT NULL,
--     updated_at BIGINT NOT NULL,
--     deleted_at BIGINT
-- );

-- Создание таблицы для хранения произвольных текстовых данных
CREATE TABLE IF NOT EXISTS text_data (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    text TEXT,
    checksum VARCHAR(64),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- Создание таблицы для хранения произвольных бинарных данных
CREATE TABLE IF NOT EXISTS binary_data (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data BYTEA, -- Nullable, так как на сервере хранятся только метаданные
    metadata TEXT,
    name VARCHAR(255),
    size BIGINT,
    checksum VARCHAR(64),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- Создание таблицы для хранения данных банковских карт
CREATE TABLE IF NOT EXISTS bank_cards (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_number_data TEXT NOT NULL,
    card_holder_data TEXT NOT NULL,
    expiry_date_data TEXT NOT NULL,
    cvc_data TEXT NOT NULL,
    metadata TEXT,
    checksum VARCHAR(64),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

-- Создание таблицы для устройств пользователя
CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name VARCHAR(255), -- Имя устройства для пользователя
    last_sync BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (user_id, id)
);

-- Создание индексов
CREATE INDEX IF NOT EXISTS idx_login_passwords_user_id ON login_passwords(user_id);
CREATE INDEX IF NOT EXISTS idx_text_data_user_id ON text_data(user_id);
CREATE INDEX IF NOT EXISTS idx_binary_data_user_id ON binary_data(user_id);
CREATE INDEX IF NOT EXISTS idx_bank_cards_user_id ON bank_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);