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

	// SQLite disables FK enforcement by default; enable it per connection.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	return db, nil
}

func InitSchema(db *sql.DB) error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return err
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM models").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = db.Exec(`
    INSERT OR IGNORE INTO models (name, input_per_million_cents, output_per_million_cents) VALUES
  ('claude-sonnet-4-20250514',  300,  1500),
  ('claude-haiku-4-20250514',    80,   400),
  ('claude-opus-4-20250514',   1500,  7500),
  ('gpt-4o',                    250,  1000),
  ('gpt-4o-mini',                15,    60),
  ('gpt-4.1',                   200,   800),
  ('gpt-4.1-mini',               40,   160),
  ('gemini-2.5-pro',            150,  1000),
  ('gemini-2.5-flash',           15,   60);
  `)
		if err != nil {
			return err
		}
	}

	return nil
}