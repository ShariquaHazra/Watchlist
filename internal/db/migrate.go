package db

import (
	"database/sql"
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(database *sql.DB) error {
	driver, err := postgres.WithInstance(database, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/db/migrations", // migrations folder
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No new migrations to run")
			return nil
		}
		return err // Duplicate version → error aayega
	}

	return nil
}
