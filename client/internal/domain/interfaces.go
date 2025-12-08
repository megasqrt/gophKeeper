package domain

import (
	models "gophKeeper/pkg/grpchelper"
	"time"
)

// Syncable определяет общие методы для всех сущностей, которые могут быть синхронизированы.
type Syncable interface {
	GetLocalID() string
	GetServerID() string
	GetChangeTime() time.Time
	ToMap() map[string]interface{}
}

// LocalStorage определяет общий интерфейс для локального хранилища,
// используемый всеми моделями TUI и сервисами.
type LocalStorage interface {
	Unlock(login, password string) error
	IsLoggedIn() bool
	IsFirstRun() bool
	LocalRegister(user, password string) error
	SaveUserCredentials(login, token, deviceID string) error
	GetUserCredentials() (login, token, deviceID string, err error)

	SaveCard(cardData *models.Card) error
	UpdateCard(cardData *models.Card) error
	GetCards() ([]models.Card, error)
	GetCardsByIDs(ids []string) ([]models.Card, error)
	GetShortCards() ([]models.SyncInfo, error)
	DeleteCard(id string) error

	SaveLastSyncTime(t time.Time) error
	GetLastSyncTime() (time.Time, error)

	SavePass(passData *models.Password) error
	UpdatePass(passData *models.Password) error
	GetPasss() ([]models.Password, error)
	GetPasswordsByIDs(ids []string) ([]models.Password, error)
	GetShortPasswords() ([]models.SyncInfo, error)
	DeletePass(id string) error

	SaveText(textData *models.TextData) error
	UpdateText(textData *models.TextData) error
	GetTexts() ([]models.TextData, error)
	GetTextsByIDs(ids []string) ([]models.TextData, error)
	GetShortTexts() ([]models.SyncInfo, error)
	DeleteText(id string) error

	GetFiles() ([]models.FileData, error)
	GetFilesByIDs(ids []string) ([]models.FileData, error)
	GetShortFiles() ([]models.SyncInfo, error)
	GetFileByID(id string) (map[string]interface{}, error)
	UpdateFile(data *models.FileData) error
	SaveFile(data *models.FileData, content []byte) error
	SaveFileMetadata(data *models.FileData) error
	DeleteFileByID(id string) error

	Close() error
}
