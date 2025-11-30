package tui

import "time"

// LocalStorage определяет общий интерфейс для локального хранилища,
// используемый всеми моделями TUI.
type LocalStorage interface {
	Unlock(login, password string) error
	IsLoggedIn() bool
	IsFirstRun() bool
	LocalRegister(user, password string) error
	SaveUserCredentials(login, token string) error
	GetUserCredentials() (user, token string, err error)

	SaveCard(cardData map[string]string) error
	UpdateCard(cardData map[string]string) error
	GetCards() ([]map[string]string, error)

	SaveLastSyncTime(t time.Time) error
	GetLastSyncTime() (time.Time, error)

	SavePass(passData map[string]string) error
	UpdatePass(passData map[string]string) error
	GetPasss() ([]map[string]string, error)

	SaveText(textData map[string]string) error
	UpdateText(textData map[string]string) error
	GetTexts() ([]map[string]string, error)
	DeleteText(id string) error

	GetFiles() ([]map[string]interface{}, error)
	GetFileByID(id string) (map[string]interface{}, error)
	SaveFile(data map[string]interface{}) error
	DeleteFileByID(id string) error

	Close() error
}

// errMsg - это общий тип для передачи ошибок в виде сообщений tea.
type errMsg error
