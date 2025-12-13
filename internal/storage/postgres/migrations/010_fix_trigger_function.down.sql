-- Откат: возвращаем старую версию триггера (но это не будет работать, так как поля уже BIGINT)
CREATE OR REPLACE FUNCTION log_login_password_history()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO login_passwords_history (id, user_id, login_data, password_data, metadata, created_at, updated_at, deleted_at)
    VALUES (OLD.id, OLD.user_id, OLD.login_data, OLD.password_data, OLD.metadata,
            (EXTRACT(EPOCH FROM OLD.created_at))::bigint,
            (EXTRACT(EPOCH FROM OLD.updated_at))::bigint,
            (EXTRACT(EPOCH FROM OLD.deleted_at))::bigint);
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

