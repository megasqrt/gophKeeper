package model

import (
	"time"

	"github.com/google/uuid"
)

// TextData представляет текстовую заметку в хранилище.
type TextData struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	Title     string     `db:"title"`
	Text      string     `db:"text"`
	Checksum  string     `db:"checksum"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}
