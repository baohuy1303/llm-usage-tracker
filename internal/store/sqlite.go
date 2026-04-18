package store

import(
	"database/sql"
	_ "modernc.org/sqlite"
	_ "embed"
)

//go:embed migrations/001_init.sql
var schemaSQL string

func NewSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1) // SQLite only supports one connection at a time

	return db, nil
}

func InitSchema(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	return err
}