package domain

import (
	"gophKeeper/client/internal/domain/model"
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

	SaveCard(cardData *model.Card) error
	UpdateCard(cardData *model.Card) error
	GetCards() ([]model.Card, error)
	DeleteCard(id string) error

	SaveLastSyncTime(t time.Time) error
	GetLastSyncTime() (time.Time, error)

	SavePass(passData *model.Password) error
	UpdatePass(passData *model.Password) error
	GetPasss() ([]model.Password, error)
	DeletePass(id string) error

	SaveText(textData *model.TextData) error
	UpdateText(textData *model.TextData) error
	GetTexts() ([]model.TextData, error)
	DeleteText(id string) error

	GetFiles() ([]model.FileData, error)
	GetFileByID(id string) (map[string]interface{}, error)
	UpdateFile(data *model.FileData) error
	SaveFile(data *model.FileData, content []byte) error
	SaveFileMetadata(data *model.FileData) error
	DeleteFileByID(id string) error

	Close() error
}
