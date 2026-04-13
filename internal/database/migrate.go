package database

import (
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) error {
	// try relative path first (local dev), fall back to absolute (Docker)
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		m, err = migrate.New("file:///migrations", databaseURL)
	}
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	slog.Info("migrations applied successfully")
	return nil
}
