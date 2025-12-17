// internal/pkg/migrate/manager.go
package migrate

import (
    "fmt"
    "log"
    
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/sqlite3"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    _ "github.com/mattn/go-sqlite3"
)

// Manager - менеджер миграций
type Manager struct {
    m *migrate.Migrate
}

// NewManager создает менеджер миграций
func NewManager(dbPath, migrationsPath string) (*Manager, error) {
    // Создаем URL для базы данных
    dbURL := fmt.Sprintf("sqlite3://%s", dbPath)
    
    // Создаем URL для миграций
    migrationsURL := fmt.Sprintf("file://%s", migrationsPath)
    
    // Создаем мигратор
    m, err := migrate.New(migrationsURL, dbURL)
    if err != nil {
        return nil, fmt.Errorf("failed to create migrate instance: %w", err)
    }
    
    return &Manager{m: m}, nil
}

// Up применяет все миграции
func (mg *Manager) Up() error {
    log.Println("Applying migrations...")
    if err := mg.m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to migrate up: %w", err)
    }
    return nil
}

// Down откатывает все миграции
func (mg *Manager) Down() error {
    log.Println("Rolling back migrations...")
    if err := mg.m.Down(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to migrate down: %w", err)
    }
    return nil
}

// Steps применяет/откатывает N миграций
// func (mg *Manager) Steps(n int) error {
//     log.Printf("Applying %d migration steps...", n)
//     if err := mg.m.Steps(n); err != nil {
//         return fmt.Errorf("failed to migrate steps: %w", err)
//     }
//     return nil
// }

// Version показывает текущую версию
// func (mg *Manager) Version() (uint, bool, error) {
//     return mg.m.Version()
// }

// // Force устанавливает конкретную версию
// func (mg *Manager) Force(version int) error {
//     log.Printf("Forcing migration version to %d...", version)
//     return mg.m.Force(version)
// }