package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the sql.DB connection
type DB struct {
	Conn *sql.DB
}

// New creates a new database connection and runs migrations
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{Conn: conn}

	if err := db.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}

// runMigrations executes the migration SQL files
func (db *DB) runMigrations() error {
	return db.runMigrationDirect()
}

// runMigrationDirect runs migration SQL directly
func (db *DB) runMigrationDirect() error {
	schema := `
CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone_number TEXT,
    email TEXT,
    linked_id INTEGER,
    link_precedence TEXT CHECK(link_precedence IN ('primary', 'secondary')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (linked_id) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_phone ON contacts(phone_number);
CREATE INDEX IF NOT EXISTS idx_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_linked_id ON contacts(linked_id);
`
	_, err := db.Conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.Conn.Close()
}
