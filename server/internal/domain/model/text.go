package model

import (
	"github.com/google/uuid"
)

// TextData представляет текстовую заметку в хранилище.
type TextData struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Title     string    `db:"title"`
	Text      string    `db:"text"`
	Checksum  string    `db:"checksum"`
	CreatedAt int64     `db:"created_at"`
	UpdatedAt int64     `db:"updated_at"`
	DeletedAt *int64    `db:"deleted_at"`
}
