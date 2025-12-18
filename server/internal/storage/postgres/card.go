package postgres

import (
	"context"
	"errors"
	"fmt"
	"gophKeeper/server/internal/domain/model"

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
func (r *CardRepository) Create(ctx context.Context, card *model.Card) (int64, error) {
	query := `
		INSERT INTO bank_cards (user_id, card_number_data, card_holder_data, expiry_date_data, cvc_data, metadata, checksum, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	var id int64
	err := r.db.GetContext(ctx, &id, query, card.UserID, card.Number, card.Holder, card.Expiry, card.CVV, card.Metadata, card.Checksum, card.CreatedAt, card.UpdatedAt, card.Version)
	if err != nil {
		return 0, fmt.Errorf("failed to insert card: %w", err)
	}
	return id, nil
}

// Update обновляет существующую карту.
func (r *CardRepository) Update(ctx context.Context, card *model.Card) error {
	query := `
		UPDATE bank_cards
		SET card_number_data = :number, card_holder_data = :holder, expiry_date_data = :expiry, cvc_data = :cvv, metadata = :metadata, checksum = :checksum, updated_at = :updated_at, version = :version
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
	query := `SELECT id, user_id, card_number_data as number, card_holder_data as holder, expiry_date_data as expiry, cvc_data as cvv, metadata, checksum, created_at, updated_at, deleted_at, version FROM bank_cards WHERE user_id = $1 AND deleted_at IS NULL ORDER BY updated_at DESC`
	err := r.db.SelectContext(ctx, &cards, query, userID)
	return cards, err
}

// Delete помечает карту как удаленную (soft delete).
func (r *CardRepository) Delete(ctx context.Context, id int64, userID uuid.UUID) error {
	query := `UPDATE bank_cards SET deleted_at = CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT) WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}

// GetDataDeviceLastSinc получает все обновленные карты с последней синхронизации.
func (r *CardRepository) GetDataDeviceLastSinc(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID) ([]*model.Card, error) {
	var cards []*model.Card
	query := `
	SELECT 
		bc.id, bc.user_id, bc.card_number_data as number, bc.card_holder_data as holder, bc.expiry_date_data as expiry, bc.cvc_data as cvv, bc.metadata, bc.checksum, bc.created_at, bc.updated_at, bc.deleted_at, bc.version 
	FROM bank_cards bc
	LEFT JOIN devices d ON bc.user_id = d.user_id
	WHERE bc.user_id = $1 AND d.id = $2 AND bc.updated_at > COALESCE(d.last_sync, 0)
	ORDER BY bc.updated_at DESC`
	err := r.db.SelectContext(ctx, &cards, query, userID, deviceID)
	return cards, err
}
