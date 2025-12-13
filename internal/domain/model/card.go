package model

import (
	"github.com/google/uuid"
)

// Card представляет данные кредитной карты в хранилище.
type Card struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Number    string    `db:"number"`
	Holder    string    `db:"holder"`
	Expiry    string    `db:"expiry"`
	CVV       string    `db:"cvv"`
	Metadata  string    `db:"metadata"`
	Checksum  string    `db:"checksum"`
	CreatedAt int64     `db:"created_at"`
	UpdatedAt int64     `db:"updated_at"`
	DeletedAt *int64    `db:"deleted_at"`
}

