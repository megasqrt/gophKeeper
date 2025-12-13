package postgres

import (
	"context"
	"database/sql"
	"errors"
	"gophKeeper/internal/domain/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TextDataRepository реализует методы для работы с текстовыми заметками в PostgreSQL.
type TextDataRepository struct {
	db *sqlx.DB
}

// NewTextDataRepository создает новый экземпляр TextDataRepository.
func NewTextDataRepository(db *sqlx.DB) *TextDataRepository {
	return &TextDataRepository{db: db}
}

// Create создает новую текстовую заметку в базе данных.
func (r *TextDataRepository) Create(ctx context.Context, data *model.TextData) error {
	query := `
		INSERT INTO text_data (id, user_id, title, text, checksum, created_at, updated_at)
		VALUES (:id, :user_id, :title, :text, :checksum, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, data)
	return err
}

// GetByID извлекает текстовую заметку по ее ID.
func (r *TextDataRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.TextData, error) {
	var data model.TextData
	query := `SELECT id, user_id, title, text, checksum, created_at, updated_at, deleted_at FROM text_data WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &data, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("text data not found")
		}
		return nil, err
	}
	return &data, nil
}

// GetByUserID извлекает все текстовые заметки для указанного пользователя.
func (r *TextDataRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.TextData, error) {
	var data []*model.TextData
	query := `SELECT id, user_id, title, text, checksum, created_at, updated_at, deleted_at FROM text_data WHERE user_id = $1 AND deleted_at IS NULL ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &data, query, userID)
	return data, err
}

// Update обновляет существующую текстовую заметку.
func (r *TextDataRepository) Update(ctx context.Context, data *model.TextData) error {
	query := `
		UPDATE text_data
		SET title = :title, text = :text, checksum = :checksum, updated_at = :updated_at
		WHERE id = :id AND user_id = :user_id
	`
	result, err := r.db.NamedExecContext(ctx, query, data)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no rows were updated, text data not found or user mismatch")
	}
	return nil
}

// Delete помечает текстовую заметку как удаленную (soft delete).
func (r *TextDataRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE text_data SET deleted_at = CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT) WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}
