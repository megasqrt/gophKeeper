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
func (r *PasswordRepository) Create(ctx context.Context, pass *model.Password) (int64, error) {
	query := `
		INSERT INTO login_passwords (user_id, login_data, password_data, metadata, checksum, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id int64
	err := r.db.GetContext(ctx, &id, query, pass.UserID, pass.Login, pass.Password, pass.Description, pass.Checksum, pass.CreatedAt, pass.UpdatedAt, pass.Version)
	if err != nil {
		return 0, fmt.Errorf("failed to insert password: %w", err)
	}
	return id, nil
}

// Update обновляет существующий пароль.
func (r *PasswordRepository) Update(ctx context.Context, pass *model.Password) error {
	query := `
		UPDATE login_passwords
		SET login_data = :login, password_data = :password, metadata = :description, checksum = :checksum, updated_at = :updated_at, version = :version
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
	query := `SELECT id, user_id, login_data as login, password_data as password, metadata as description, checksum, created_at, updated_at, deleted_at, version FROM login_passwords WHERE user_id = $1 AND deleted_at IS NULL ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &passwords, query, userID)
	return passwords, err
}

// Delete помечает пароль как удаленный.
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

// FindByChecksum находит пароль по checksum и userID для предотвращения дубликатов.
func (r *PasswordRepository) FindByChecksum(ctx context.Context, userID uuid.UUID, checksum string) (*model.Password, error) {
	var pass model.Password
	query := `SELECT id, user_id, login_data as login, password_data as password, metadata as description, checksum, created_at, updated_at, deleted_at, version 
	          FROM login_passwords 
	          WHERE user_id = $1 AND checksum = $2 AND deleted_at IS NULL 
	          LIMIT 1`
	err := r.db.GetContext(ctx, &pass, query, userID, checksum)
	if err != nil {
		return nil, err
	}
	return &pass, nil
}

// GetDataDeviceLastSinc получает все обновленные пароли с последней синхронизации.
func (r *PasswordRepository) GetDataDeviceLastSinc(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID) ([]*model.Password, error) {
	var passwords []*model.Password
	query := `
	SELECT 
		lp.id, lp.user_id, lp.login_data as login, lp.password_data as password, lp.metadata as description, lp.checksum, lp.created_at, lp.updated_at, lp.deleted_at, lp.version 
	FROM login_passwords lp
	LEFT JOIN devices d ON lp.user_id = d.user_id
	WHERE lp.user_id = $1 AND d.id = $2 AND lp.updated_at > COALESCE(d.last_sync, 0)
	ORDER BY lp.updated_at DESC`
	err := r.db.SelectContext(ctx, &passwords, query, userID, deviceID)
	return passwords, err
}
