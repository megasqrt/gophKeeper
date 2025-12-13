package postgres

import (
	"context"
	"errors"
	"gophKeeper/internal/domain/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CardRepository реализует методы для работы с картами в PostgreSQL.
type CardRepository struct {
	db *sqlx.DB
}

// NewCardRepository создает новый экземпляр CardRepository.
func NewCardRepository(db *sqlx.DB) *CardRepository {
	return &CardRepository{db: db}
}

// Create создает новую карту в базе данных.
func (r *CardRepository) Create(ctx context.Context, card *model.Card) error {
	query := `
		INSERT INTO bank_cards (id, user_id, card_number_data, card_holder_data, expiry_date_data, cvc_data, metadata, checksum, created_at, updated_at)
		VALUES (:id, :user_id, :number, :holder, :expiry, :cvv, :metadata, :checksum, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, card)
	return err
}

// Update обновляет существующую карту.
func (r *CardRepository) Update(ctx context.Context, card *model.Card) error {
	query := `
		UPDATE bank_cards
		SET card_number_data = :number, card_holder_data = :holder, expiry_date_data = :expiry, cvc_data = :cvv, metadata = :metadata, checksum = :checksum, updated_at = :updated_at
		WHERE id = :id AND user_id = :user_id
	`
	result, err := r.db.NamedExecContext(ctx, query, card)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no rows were updated, card not found or user mismatch")
	}
	return nil
}

// GetByUserID извлекает все карты для указанного пользователя.
func (r *CardRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Card, error) {
	var cards []*model.Card
	query := `SELECT id, user_id, card_number_data as number, card_holder_data as holder, expiry_date_data as expiry, cvc_data as cvv, metadata, checksum, created_at, updated_at, deleted_at FROM bank_cards WHERE user_id = $1 AND deleted_at IS NULL ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &cards, query, userID)
	return cards, err
}

// Delete помечает карту как удаленную (soft delete).
func (r *CardRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE bank_cards SET deleted_at = EXTRACT(EPOCH FROM NOW())::bigint WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}
