package domain

import (
	models "gophKeeper/pkg/grpchelper"
)

// Syncable определяет общие методы для всех сущностей, которые могут быть синхронизированы.
type Syncable interface {
	GetLocalID() string
	GetServerID() string

	GetChangeTime() int64
	ToMap() map[string]interface{}
}

// LocalStorage определяет общий интерфейс для локального хранилища,
// используемый всеми моделями TUI и сервисами.
type LocalStorage interface {
	Unlock(login, password string) error
	IsLoggedIn() bool
	IsFirstRun() bool
	LocalRegister(user, password string) error
	SaveUserCredentials(login, token, deviceID string, encryptedMasterKey []byte) error
	GetUserCredentials() (login, token, deviceID string, encryptedMasterKey []byte, err error)
	//InitializeEncryptor(password string) error                                           // Инициализирует Encryptor из сохраненного мастер-ключа (password может быть пустым, тогда будет получен из хранилища)
	//InitializeEncryptorWithData(password, login string, encryptedMasterKey []byte) error // Инициализирует Encryptor с уже полученными данными (избегает повторного вызова GetUserCredentials)

	SaveCard(cardData *models.Card) error
	UpdateCard(cardData *models.Card) error
	GetCards() ([]models.Card, error)
	GetCardsByIDs(ids []int64) ([]models.Card, error)

	DeleteCard(id int64) error

	SaveLastSyncTime(t int64) error
	GetLastSyncTime() (int64, error)

	SavePass(passData *models.Password) error
	UpdatePass(passData *models.Password) error
	//UpdatePasswords(passData *[]models.Password) error
	GetPasswords() ([]models.Password, error)
	DeletePass(id int64) error

	SaveText(textData *models.TextData) error
	UpdateText(textData *models.TextData) error
	GetTexts() ([]models.TextData, error)
	GetTextsByIDs(ids []int64) ([]models.TextData, error)

	DeleteText(id int64) error

	GetFiles() ([]models.FileData, error)
	GetFilesByIDs(ids []int64) ([]models.FileData, error)

	GetFileByID(id int64) (map[string]interface{}, error)
	UpdateFile(data *models.FileData) error
	SaveFile(data *models.FileData, content []byte) error
	SaveFileMetadata(data *models.FileData) error
	DeleteFileByID(id int64) error

	Close() error
}
