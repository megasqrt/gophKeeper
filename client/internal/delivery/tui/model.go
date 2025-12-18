package tui

import (
	"context"
)

// Syncer определяет интерфейс для сервиса синхронизации.
type Syncer interface {
	Sync(ctx context.Context) error
}

// errMsg - это общий тип для передачи ошибок в виде сообщений tea.
type errMsg error
