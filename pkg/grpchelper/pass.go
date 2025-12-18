package grpchelper

import "strconv"

// Password представляет собой данные пароля.
type Password struct {
	LocalID     int64
	ServerID    int64

	Login       string
	Password    string
	Description string
	
	Checksum    string
	
	CreateTime  int64
	ChangeTime  int64
    Version 	int32

	Deleted     bool
}


func (p Password) GetChangeTime() int64 { return p.ChangeTime }
func (p Password) GetLocalID() int64       { return p.LocalID }
func (p Password) GetDeleted() bool         { return p.Deleted }
func (p *Password) SetDeleted(deleted bool) { p.Deleted = deleted }
func (p Password) GetServerID() int64      { return p.ServerID }
func (p Password) GetServerIDString() string { return strconv.FormatInt(p.ServerID, 10) }
