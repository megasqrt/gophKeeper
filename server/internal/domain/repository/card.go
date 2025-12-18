package repository

import (
	"context"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
)

// CardRepository определяет интерфейс для работы с картами
type CardRepository interface {
	Create(ctx context.Context, card *model.Card) (int64, error)
	Update(ctx context.Context, card *model.Card) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Card, error)
	Delete(ctx context.Context, id int64, userID uuid.UUID) error
	GetDataDeviceLastSinc(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID) ([]*model.Card, error)
}
