package migrations

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type migration struct {
	dsn string
}

func NewMigration(dsn string) *migration {
	return &migration{dsn: dsn}
}

func (m *migration) Up(sourceURL string) error {

	mgr, err := migrate.New(sourceURL, m.dsn)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer mgr.Close()

	if err := mgr.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	return nil
}
