-- Создание таблицы конфигурации пользователя
CREATE TABLE IF NOT EXISTS config (
    login TEXT,    
    user TEXT PRIMARY KEY NOT NULL,     
    password_hash TEXT NOT NULL,
    token TEXT,
    device TEXT,
    last_sync INTEGER,  -- SQLite хранит как INTEGER
    encrypted_master_key BLOB
);

CREATE TABLE IF NOT EXISTS credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER,
    version INTEGER,
    data BLOB NOT NULL,              -- Зашифрованный JSON всей модели Password
    checksum TEXT,                   -- Для синхронизации
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER
);

CREATE TABLE IF NOT EXISTS cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER,
    version INTEGER,
    data BLOB NOT NULL,              -- Зашифрованный JSON всей модели Card
    checksum TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER
);

CREATE TABLE IF NOT EXISTS note (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER,
    title TEXT NOT NULL,
    data BLOB NOT NULL,              -- Зашифрованный JSON всей модели TextData
    checksum TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER,
    version INTEGER
);


-- binary_data с отдельными колонками для метаданных файла
CREATE TABLE IF NOT EXISTS binary_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER,
    name TEXT NOT NULL,              -- Имя файла
    data BLOB NOT NULL,              -- Зашифрованное содержимое файла
    size int64,                   -- Зашифрованный JSON метаданных FileData (без содержимого)
    checksum TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER,
    version INTEGER
);
-- Индексы остаются теми же
CREATE INDEX IF NOT EXISTS idx_credentials_deleted_at ON credentials(deleted_at);
CREATE INDEX IF NOT EXISTS idx_credentials_updated_at ON credentials(updated_at);
CREATE INDEX IF NOT EXISTS idx_cards_deleted_at ON cards(deleted_at);
CREATE INDEX IF NOT EXISTS idx_cards_updated_at ON cards(updated_at);
CREATE INDEX IF NOT EXISTS idx_note_deleted_at ON note(deleted_at);
CREATE INDEX IF NOT EXISTS idx_note_updated_at ON note(updated_at);
CREATE INDEX IF NOT EXISTS idx_binary_data_deleted_at ON binary_data(deleted_at);
CREATE INDEX IF NOT EXISTS idx_binary_data_updated_at ON binary_data(updated_at);
