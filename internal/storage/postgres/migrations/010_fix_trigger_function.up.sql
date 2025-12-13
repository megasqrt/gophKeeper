-- Исправляем триггер log_login_password_history
-- После миграции 002 все поля времени уже являются BIGINT, поэтому не нужно извлекать EPOCH
CREATE OR REPLACE FUNCTION log_login_password_history()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO login_passwords_history (id, user_id, login_data, password_data, metadata, created_at, updated_at, deleted_at)
    VALUES (OLD.id, OLD.user_id, OLD.login_data, OLD.password_data, OLD.metadata,
            OLD.created_at,
            OLD.updated_at,
            OLD.deleted_at);
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

