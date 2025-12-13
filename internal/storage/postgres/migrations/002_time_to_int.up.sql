-- Alter users table
ALTER TABLE users
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN created_at TYPE BIGINT USING (EXTRACT(EPOCH FROM created_at))::bigint,
    ALTER COLUMN updated_at DROP DEFAULT,
    ALTER COLUMN updated_at TYPE BIGINT USING (EXTRACT(EPOCH FROM updated_at))::bigint;

-- Alter login_passwords table
ALTER TABLE login_passwords
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN created_at TYPE BIGINT USING (EXTRACT(EPOCH FROM created_at))::bigint,
    ALTER COLUMN updated_at DROP DEFAULT,
    ALTER COLUMN updated_at TYPE BIGINT USING (EXTRACT(EPOCH FROM updated_at))::bigint,
    ALTER COLUMN deleted_at TYPE BIGINT USING (EXTRACT(EPOCH FROM deleted_at))::bigint;

-- Alter login_passwords_history table
-- This table is a bit different, it doesn't have defaults
ALTER TABLE login_passwords_history
    ALTER COLUMN created_at TYPE BIGINT USING (EXTRACT(EPOCH FROM created_at))::bigint,
    ALTER COLUMN updated_at TYPE BIGINT USING (EXTRACT(EPOCH FROM updated_at))::bigint,
    ALTER COLUMN deleted_at TYPE BIGINT USING (EXTRACT(EPOCH FROM deleted_at))::bigint;


-- Alter text_data table
ALTER TABLE text_data
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN created_at TYPE BIGINT USING (EXTRACT(EPOCH FROM created_at))::bigint,
    ALTER COLUMN updated_at DROP DEFAULT,
    ALTER COLUMN updated_at TYPE BIGINT USING (EXTRACT(EPOCH FROM updated_at))::bigint,
    ALTER COLUMN deleted_at TYPE BIGINT USING (EXTRACT(EPOCH FROM deleted_at))::bigint;

-- Alter binary_data table
ALTER TABLE binary_data
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN created_at TYPE BIGINT USING (EXTRACT(EPOCH FROM created_at))::bigint,
    ALTER COLUMN updated_at DROP DEFAULT,
    ALTER COLUMN updated_at TYPE BIGINT USING (EXTRACT(EPOCH FROM updated_at))::bigint,
    ALTER COLUMN deleted_at TYPE BIGINT USING (EXTRACT(EPOCH FROM deleted_at))::bigint;

-- Alter bank_cards table
ALTER TABLE bank_cards
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN created_at TYPE BIGINT USING (EXTRACT(EPOCH FROM created_at))::bigint,
    ALTER COLUMN updated_at DROP DEFAULT,
    ALTER COLUMN updated_at TYPE BIGINT USING (EXTRACT(EPOCH FROM updated_at))::bigint,
    ALTER COLUMN deleted_at TYPE BIGINT USING (EXTRACT(EPOCH FROM deleted_at))::bigint;

-- Alter devices table
ALTER TABLE devices
    ALTER COLUMN last_sync_at TYPE BIGINT USING (EXTRACT(EPOCH FROM last_sync_at))::bigint,
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN created_at TYPE BIGINT USING (EXTRACT(EPOCH FROM created_at))::bigint,
    ALTER COLUMN updated_at DROP DEFAULT,
    ALTER COLUMN updated_at TYPE BIGINT USING (EXTRACT(EPOCH FROM updated_at))::bigint;

-- Re-create the trigger function to handle the new data types.
-- After migration 002, all timestamp columns are already BIGINT, so we don't need to extract EPOCH.
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
