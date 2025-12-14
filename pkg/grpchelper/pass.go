package grpchelper

// Password представляет собой данные пароля.
type Password struct {
	LocalID     string
	ServerID    string

	Login       string
	Password    string
	Description string
	
	Checksum    string
	
	CreateTime  int64
	ChangeTime  int64
	SyncTime    int64
	ClientVersion int64 
    ServerVersion int64

	Deleted     bool
}

func (p Password) GetLocalID() string    { return p.LocalID }
func (p *Password) SetLocalID(id string) { p.LocalID = id }

func (p Password) GetServerID() string    { return p.ServerID }
func (p *Password) SetServerID(id string) { p.ServerID = id }

func (p Password) GetChangeTime() int64 { return p.ChangeTime }

func (p Password) GetDeleted() bool         { return p.Deleted }
func (p *Password) SetDeleted(deleted bool) { p.Deleted = deleted }
