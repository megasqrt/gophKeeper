package grpchelper

import "time"

type TextData struct {
	LocalID    string
	ServerID   string
	Title      string
	Text       string
	Checksum   string
	ChangeTime time.Time
	SyncTime   time.Time
	Deleted    bool
}

func (t TextData) GetLocalID() string       { return t.LocalID }
func (t TextData) GetServerID() string      { return t.ServerID }
func (t TextData) GetChangeTime() time.Time { return t.ChangeTime }
func (t *TextData) SetLocalID(id string)    { t.LocalID = id }

func (i TextData) FilterValue() string { return i.Title }

// ToMap converts a TextData model to a map for storage.
func (t *TextData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         t.LocalID, // Хранилище использует "id" для LocalID
		"server_id":  t.ServerID,
		"title":      t.Title,
		"text":       t.Text,
		"checksum":   t.Checksum,
		"changeTime": t.ChangeTime.Format(time.RFC3339Nano),
		"syncTime":   t.SyncTime.Format(time.RFC3339Nano),
		"deleted":    t.Deleted,
	}
}
