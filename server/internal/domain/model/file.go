package model

import (
	"github.com/google/uuid"
)

// File представляет метаданные файла в хранилище.
type File struct {
	ID        int64     `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Name      string    `db:"name"`
	Metadata  string    `db:"metadata"`
	Size      int64     `db:"size"`
	Checksum  string    `db:"checksum"`
	CreatedAt int64     `db:"created_at"`
	UpdatedAt int64     `db:"updated_at"`
	DeletedAt *int64    `db:"deleted_at"`
	Version   int32     `db:"version"`
}

