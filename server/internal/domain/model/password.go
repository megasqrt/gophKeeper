package model

import (
	"github.com/google/uuid"
)

// Password представляет данные пароля в хранилище.
type Password struct {
	ID          uuid.UUID `db:"id"`
	UserID      uuid.UUID `db:"user_id"`
	Login       string    `db:"login"`
	Password    string    `db:"password"`
	Description string    `db:"description"`
	Checksum    string    `db:"checksum"`
	CreatedAt    int64     `db:"created_at"`
	UpdatedAt   int64     `db:"updated_at"`
	DeletedAt   *int64    `db:"deleted_at"`
}

