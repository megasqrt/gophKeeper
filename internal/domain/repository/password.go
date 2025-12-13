package repository

import (
	"context"
	"gophKeeper/internal/domain/model"

	"github.com/google/uuid"
)

// PasswordRepository определяет интерфейс для работы с паролями
type PasswordRepository interface {
	Create(ctx context.Context, pass *model.Password) error
	Update(ctx context.Context, pass *model.Password) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Password, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

