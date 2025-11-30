package model

import "time"

// Password представляет собой данные пароля.
type Password struct {
	ID          string
	Login       string
	Password    string
	Description string
	ChangeTime  time.Time
	SyncTime    time.Time
}