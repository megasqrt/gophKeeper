package model

import "time"

// FileData представляет метаданные файла.
type FileData struct {
	LocalID    string
	ServerID   string
	Name       string
	Metadata   string
	Size       int64
	ChangeTime time.Time
	SyncTime   time.Time
	Deleted    bool
}

func (f FileData) GetLocalID() string       { return f.LocalID }
func (f FileData) GetServerID() string      { return f.ServerID }
func (f FileData) GetChangeTime() time.Time { return f.ChangeTime }
func (f *FileData) SetLocalID(id string)    { f.LocalID = id }
