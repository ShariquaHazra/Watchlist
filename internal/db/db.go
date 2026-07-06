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
		log.Fatal("Database open error:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping error:", err)
	}

	log.Println("Database connected successfully")

	// Run migrations automatically
	err = RunMigrations(connStr)
	if err != nil {
		log.Printf("Migration failed: %v", err)

		// optional:
		// comment below if you want app to stop on migration error
		// log.Fatal(err)

	} else {
		log.Println("Migrations completed successfully")
	}

	return db
}