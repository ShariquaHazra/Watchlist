package db

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(connStr string) error {

	m, err := migrate.New(
		"file://./internal/db/migrations",
		connStr,
	)

	if err != nil {
		return err
	}

	err = m.Up()

	if err != nil {

		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No new migrations to run")
			return nil
		}

		return err
	}

	return nil
}