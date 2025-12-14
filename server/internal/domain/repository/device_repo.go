package repository

import (
	"context"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
)

// DeviceRepository определяет интерфейс для работы с хранилищем устройств.
type DeviceRepository interface {
	Create(ctx context.Context, device *model.Device) error
	Update(ctx context.Context, device *model.Device) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Device, error)
	FindByID(ctx context.Context, deviceID uuid.UUID) (*model.Device, error)
	Delete(ctx context.Context, deviceID uuid.UUID) error
}
