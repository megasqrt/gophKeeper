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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    device_id VARCHAR(255) NOT NULL, -- Уникальный идентификатор устройства
    device_name VARCHAR(255), -- Имя устройства для пользователя
    last_sync_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE (user_id, device_id)
);

-- Создание индексов
CREATE INDEX IF NOT EXISTS idx_login_passwords_user_id ON login_passwords(user_id);
CREATE INDEX IF NOT EXISTS idx_text_data_user_id ON text_data(user_id);
CREATE INDEX IF NOT EXISTS idx_binary_data_user_id ON binary_data(user_id);
CREATE INDEX IF NOT EXISTS idx_bank_cards_user_id ON bank_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
-- CREATE INDEX IF NOT EXISTS idx_login_passwords_history_id ON login_passwords_history(id);

-- Создание функции-триггера для сохранения истории
-- CREATE OR REPLACE FUNCTION log_login_password_history()
-- RETURNS TRIGGER AS $$
-- BEGIN
--     INSERT INTO login_passwords_history (id, user_id, login_data, password_data, metadata, created_at, updated_at, deleted_at)
--     VALUES (OLD.id, OLD.user_id, OLD.login_data, OLD.password_data, OLD.metadata,
--             OLD.created_at,
--             OLD.updated_at,
--             OLD.deleted_at);
--     RETURN OLD;
-- END;
-- $$ LANGUAGE plpgsql;

-- Применение триггера к таблице login_passwords
-- Триггер срабатывает перед UPDATE или DELETE
-- DROP TRIGGER IF EXISTS login_passwords_history_trigger ON login_passwords;
-- CREATE TRIGGER login_passwords_history_trigger
-- BEFORE UPDATE OR DELETE ON login_passwords
-- FOR EACH ROW
-- EXECUTE FUNCTION log_login_password_history();
