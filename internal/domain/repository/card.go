package repository

import (
	"context"
	"gophKeeper/internal/domain/model"

	"github.com/google/uuid"
)

// CardRepository определяет интерфейс для работы с картами
type CardRepository interface {
	Create(ctx context.Context, card *model.Card) error
	Update(ctx context.Context, card *model.Card) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Card, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

