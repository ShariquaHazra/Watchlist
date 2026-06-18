package db

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(connStr string) *sql.DB {

	if connStr == "" {
		log.Fatal("DATABASE_URL is missing")
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Database connected successfully")

	// AUTO RUN MIGRATIONS
	if err := RunMigrations(db); err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Migrations completed successfully")

	return db
}