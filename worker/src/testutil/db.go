package testutil

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	database := os.Getenv("POSTGRES_DB")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(user),
		url.QueryEscape(password),
		host,
		port,
		database,
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TruncateJobs(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`TRUNCATE jobs RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate failed: %v", err)
	}
}
