package repository

import (
	"context"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
)

// FileRepository определяет интерфейс для работы с файлами
type FileRepository interface {
	Create(ctx context.Context, file *model.File) error
	Update(ctx context.Context, file *model.File) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.File, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
