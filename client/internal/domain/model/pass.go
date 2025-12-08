package model

import "time"

// Password представляет собой данные пароля.
type Password struct {
	LocalID     string
	ServerID    string
	Login       string
	Password    string
	Description string
	ChangeTime  time.Time
	SyncTime    time.Time
	Deleted     bool
}

func (p Password) GetLocalID() string       { return p.LocalID }
func (p Password) GetServerID() string      { return p.ServerID }
func (p Password) GetChangeTime() time.Time { return p.ChangeTime }
func (p *Password) SetLocalID(id string)    { p.LocalID = id }
