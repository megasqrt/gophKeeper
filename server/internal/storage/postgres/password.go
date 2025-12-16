package postgres

import (
	"context"
	"errors"
	"fmt"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PasswordRepository реализует методы для работы с паролями в PostgreSQL.
type PasswordRepository struct {
	db *sqlx.DB
}

// NewPasswordRepository создает новый экземпляр PasswordRepository.
func NewPasswordRepository(db *sqlx.DB) *PasswordRepository {
	return &PasswordRepository{db: db}
}

// Create создает новый пароль в базе данных.
func (r *PasswordRepository) Create(ctx context.Context, pass *model.Password) (int64,error) {
	query := `
		INSERT INTO login_passwords (user_id, login_data, password_data, metadata, checksum, created_at, updated_at)
		VALUES (:user_id, :login, :password, :description, :checksum, :created_at, :updated_at)
	`
	result, err := r.db.NamedExecContext(ctx, query, pass)
    if err != nil {
        return 0, fmt.Errorf("failed to insert password: %w", err)
    }
    
    // Получаем ID последней вставленной записи
    lastInsertID, err := result.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("failed to get last insert ID: %w", err)
    }
	return lastInsertID, nil
}

// Update обновляет существующий пароль.
func (r *PasswordRepository) Update(ctx context.Context, pass *model.Password) error {
	query := `
		UPDATE login_passwords
		SET login_data = :login, password_data = :password, metadata = :description, checksum = :checksum, updated_at = :updated_at
		WHERE id = :id AND user_id = :user_id
	`
	result, err := r.db.NamedExecContext(ctx, query, pass)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no rows were updated, password not found or user mismatch")
	}
	return nil
}

// GetByUserID извлекает все пароли для указанного пользователя.
func (r *PasswordRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Password, error) {
	var passwords []*model.Password
	query := `SELECT id, user_id, login_data as login, password_data as password, metadata as description, checksum, created_at, updated_at, deleted_at, version FROM login_passwords WHERE user_id = $1 ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &passwords, query, userID)
	return passwords, err
}

// Delete помечает пароль как удаленный (soft delete).
func (r *PasswordRepository) Delete(ctx context.Context, id int64, userID uuid.UUID) error {
	query := `UPDATE login_passwords SET deleted_at = CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT) WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no rows were updated, password not found or user mismatch")
	}
	return nil
}

//получаем все обновленные пароли споследней синхронизации
func (r *PasswordRepository) GetUserDeviceLastSinc(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID) ([]*model.Password, error) {
	var passwords []*model.Password
	query := `
	SELECT 
		id, user_id, login_data as login, password_data as password, metadata as description, checksum, created_at, updated_at, deleted_at, version 
	FROM login_passwords lp
	LEFT JOIN devices ON login_passwords.user_id = devices.user_id
	WHERE user_id = $1 and devices.id = $2 and lp.updated_at > devices.last_sync
	ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &passwords, query, userID,deviceID)
	return passwords, err
}