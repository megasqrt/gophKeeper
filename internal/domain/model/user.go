package model

import (
	"time"

	"github.com/google/uuid"
)

// План реализации:
// 1. Определить модели данных, такие как User, Secret, и т.д.

// User представляет пользователя в системе.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Login        string    `json:"login" db:"login"`
	PasswordHash string    `json:"-" db:"password_hash"` // Хеш пароля не должен отправляться клиенту
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
