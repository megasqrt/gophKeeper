package repository

import (
	"context"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
)

// TextDataRepository определяет интерфейс для работы с текстовыми заметками
type TextDataRepository interface {
	Create(ctx context.Context, note *model.TextData) (int64, error)
	Update(ctx context.Context, note *model.TextData) error
	GetByID(ctx context.Context, id int64) (*model.TextData, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.TextData, error)
	Delete(ctx context.Context, id int64, userID uuid.UUID) error
}
