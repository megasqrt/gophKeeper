package tui

import "time"

// LocalStorage определяет общий интерфейс для локального хранилища,
// используемый всеми моделями TUI.
type LocalStorage interface {
	Unlock(login, password string) error
	SaveUserCredentials(login, token string) error
	GetUserCredentials() (login, token string, err error)
	SaveLastSyncTime(t time.Time) error
	GetLastSyncTime() (time.Time, error)
	IsLoggedIn() bool
	Close() error
}

// errMsg - это общий тип для передачи ошибок в виде сообщений tea.
type errMsg error
