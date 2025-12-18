package model

import (
	"github.com/google/uuid"
)

type Device struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	DeviceName string    `db:"device_name"`
	LastSync   *int64    `db:"last_sync"`
	CreatedAt  int64     `db:"created_at"`
	UpdatedAt  int64     `db:"updated_at"`
}
