package model

import (
	"github.com/google/uuid"
)

type Device struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	DeviceID   string    `db:"device_id"`
	DeviceName string    `db:"device_name"`
	CreatedAt  int64 `db:"created_at"`
	UpdatedAt  int64 `db:"updated_at"`
}
