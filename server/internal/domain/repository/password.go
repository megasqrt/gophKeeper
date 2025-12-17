package repository

import (
	"context"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
)

// PasswordRepository определяет интерфейс для работы с паролями
type PasswordRepository interface {
	Create(ctx context.Context, pass *model.Password) (int64, error)
	Update(ctx context.Context, pass *model.Password) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Password, error)
	Delete(ctx context.Context, id int64, userID uuid.UUID) error
	//GetByServerIDs(ctx context.Context, userID uuid.UUID, serverIDs []int64) ([]*model.Password, error)
	GetUserDeviceLastSinc(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID) ([]*model.Password, error)
}
