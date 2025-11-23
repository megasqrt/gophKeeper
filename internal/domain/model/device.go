package model

import (
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	DeviceID   string    `db:"device_id"`
	DeviceName string    `db:"device_name"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
