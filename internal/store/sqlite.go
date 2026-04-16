package store

import(
	"database/sql"
	_ "modernc.org/sqlite"
)

func NewSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1) // SQLite only supports one connection at a time

	return db, nil
}

func InitSchema(db *sql.DB) error {
    query := `
    CREATE TABLE IF NOT EXISTS projects (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE,
        budget INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

    _, err := db.Exec(query)
    return err
}