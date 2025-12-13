package postgres

import (
	"context"
	"errors"
	"gophKeeper/internal/domain/model"

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
func (r *PasswordRepository) Create(ctx context.Context, pass *model.Password) error {
	query := `
		INSERT INTO login_passwords (id, user_id, login_data, password_data, metadata, checksum, created_at, updated_at)
		VALUES (:id, :user_id, :login, :password, :description, :checksum, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, pass)
	return err
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
	query := `SELECT id, user_id, login_data as login, password_data as password, metadata as description, checksum, created_at, updated_at, deleted_at FROM login_passwords WHERE user_id = $1 AND deleted_at IS NULL ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &passwords, query, userID)
	return passwords, err
}

// Delete помечает пароль как удаленный (soft delete).
func (r *PasswordRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE login_passwords SET deleted_at = EXTRACT(EPOCH FROM NOW())::bigint WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}
