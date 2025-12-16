package grpchelper

// FileData представляет метаданные файла.
type FileData struct {
	LocalID    int64
	ServerID   int64
	Name       string
	Metadata   string
	Size       int64
	Checksum   string
	ChangeTime int64
	Deleted    bool
	Version		int32
}

func (f FileData) GetLocalID() int64       { return f.LocalID }
func (f FileData) GetServerID() int64      { return f.ServerID }
func (f FileData) GetChangeTime() int64     { return f.ChangeTime }
func (f *FileData) SetLocalID(id int64)    { f.LocalID = id }
func (f FileData) GetDeleted() bool         { return f.Deleted }
func (f *FileData) SetDeleted(deleted bool) { f.Deleted = deleted }
