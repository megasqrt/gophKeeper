package repository

import (
	"context"

	"gophKeeper/server/internal/domain/model"
)

// План реализации:
// 1. Определить интерфейс репозитория для работы с пользователями.
// 2. Реализовать методы для создания, чтения, обновления и удаления пользователей.

// UserRepository определяет интерфейс для работы с хранилищем пользователей.
type UserRepository interface {
	// Create создает нового пользователя в хранилище.
	Create(ctx context.Context, user *model.User) error
	// FindByLogin находит пользователя по его логину.
	FindByLogin(ctx context.Context, login string) (*model.User, error)
	// Update обновляет данные существующего пользователя.
	Update(ctx context.Context, user *model.User) error
}
