package model

import "time"

// FileData представляет метаданные файла.
type FileData struct {
	ID         string
	Name       string
	Metadata   string
	Size       int64
	ChangeTime time.Time
	SyncTime   time.Time
}
