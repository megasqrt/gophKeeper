package postgres

import (
	"context"
	"errors"
	"gophKeeper/internal/domain/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FileRepository реализует методы для работы с файлами в PostgreSQL.
type FileRepository struct {
	db *sqlx.DB
}

// NewFileRepository создает новый экземпляр FileRepository.
func NewFileRepository(db *sqlx.DB) *FileRepository {
	return &FileRepository{db: db}
}

// Create создает новый файл в базе данных.
// Note: поле data оставляем NULL, так как мы храним только метаданные на сервере
func (r *FileRepository) Create(ctx context.Context, file *model.File) error {
	query := `
		INSERT INTO binary_data (id, user_id, name, metadata, size, checksum, data, created_at, updated_at)
		VALUES (:id, :user_id, :name, :metadata, :size, :checksum, NULL, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, file)
	return err
}

// Update обновляет существующий файл.
func (r *FileRepository) Update(ctx context.Context, file *model.File) error {
	query := `
		UPDATE binary_data
		SET name = :name, metadata = :metadata, size = :size, checksum = :checksum, updated_at = :updated_at
		WHERE id = :id AND user_id = :user_id
	`
	result, err := r.db.NamedExecContext(ctx, query, file)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no rows were updated, file not found or user mismatch")
	}
	return nil
}

// GetByUserID извлекает все файлы для указанного пользователя.
func (r *FileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.File, error) {
	var files []*model.File
	query := `SELECT id, user_id, name, metadata, size, checksum, created_at, updated_at, deleted_at FROM binary_data WHERE user_id = $1 AND deleted_at IS NULL ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &files, query, userID)
	return files, err
}

// Delete помечает файл как удаленный (soft delete).
func (r *FileRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE binary_data SET deleted_at = EXTRACT(EPOCH FROM NOW())::bigint WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}
