package postgres

import (
	"context"
	"errors"
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

// GetByUserID извлекает все пароли для указанного пользователя.
func (r *PasswordRepository) GetDeletedByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Password, error) {
	var passwords []*model.Password
	query := `SELECT id, user_id, login_data as login, password_data as password, metadata as description, checksum, created_at, updated_at, deleted_at FROM login_passwords WHERE user_id = $1 AND deleted_at IS NOT NULL ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &passwords, query, userID)
	return passwords, err
}

// GetServerIDsNotInList возвращает ID серверных записей, которых нет в списке excludeIDs.
func (r *PasswordRepository) GetServerIDsNotInList(ctx context.Context, userID uuid.UUID, excludeIDs []string) ([]string, error) {
	var ids []string
	var query string
	var err error

	if len(excludeIDs) == 0 {
		// Если список исключений пуст, возвращаем все ID
		query = `SELECT id::text FROM login_passwords WHERE user_id = $1 AND deleted_at IS NULL`
		err = r.db.SelectContext(ctx, &ids, query, userID)
	} else {
		// Используем NOT с ANY для проверки, что ID не входит в список
		query = `SELECT id::text FROM login_passwords WHERE user_id = $1 AND deleted_at IS NULL AND NOT (id::text = ANY($2::text[]))`
		err = r.db.SelectContext(ctx, &ids, query, userID, excludeIDs)
	}

	if err != nil {
		return nil, err
	}
	return ids, nil
}

// Delete помечает пароль как удаленный (soft delete).
func (r *PasswordRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
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

// GetByServerIDs извлекает пароли по их серверным ID для указанного пользователя (включая удаленные).
func (r *PasswordRepository) GetByServerIDs(ctx context.Context, userID uuid.UUID, serverIDs []string) ([]*model.Password, error) {
	if len(serverIDs) == 0 {
		return nil, nil
	}

	var passwords []*model.Password
	query, args, err := sqlx.In(`
		SELECT id, user_id, login_data as login, password_data as password, metadata as description, checksum, created_at, updated_at, deleted_at 
		FROM login_passwords 
		WHERE user_id = ? AND id::text IN (?)
	`, userID, serverIDs)
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)
	err = r.db.SelectContext(ctx, &passwords, query, args...)
	return passwords, err
}
