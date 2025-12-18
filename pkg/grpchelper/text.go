package grpchelper

type TextData struct {
	LocalID    int64
	ServerID   int64
	Title      string
	Data       string
	Checksum   string
	ChangeTime int64
	Deleted    bool
	Version    int32
}

func (t TextData) GetLocalID() int64       { return t.LocalID }
func (t TextData) GetServerID() int64      { return t.ServerID }
func (t TextData) GetChangeTime() int64     { return t.ChangeTime }
func (t *TextData) SetLocalID(id int64)    { t.LocalID = id }
func (t TextData) GetDeleted() bool         { return t.Deleted }
func (t *TextData) SetDeleted(deleted bool) { t.Deleted = deleted }

func (i TextData) FilterValue() string { return i.Title }

// ToMap converts a TextData model to a map for storage.
func (t *TextData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         t.LocalID, // Хранилище использует "id" для LocalID
		"server_id":  t.ServerID,
		"title":      t.Title,
		"data":       t.Data,
		"checksum":   t.Checksum,
		"changeTime": t.ChangeTime,
		"deleted":    t.Deleted,
		"version":    t.Version,
	}
}
