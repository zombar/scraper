package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gurgeh/scraper/models"
)

// DB wraps the database connection and provides data access methods
type DB struct {
	conn *sql.DB
}

// Config contains database configuration
type Config struct {
	Driver string
	DSN    string
}

// DefaultConfig returns a default SQLite configuration
func DefaultConfig() Config {
	return Config{
		Driver: "sqlite",
		DSN:    "scraper.db",
	}
}

// New creates a new database connection
func New(config Config) (*DB, error) {
	conn, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	db := &DB{conn: conn}

	// Run migrations
	if err := Migrate(conn); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// SaveScrapedData saves scraped data to the database
func (db *DB) SaveScrapedData(data *models.ScrapedData) error {
	// Serialize the data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Insert or replace
	query := `
		INSERT INTO scraped_data (id, url, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			id = excluded.id,
			data = excluded.data,
			updated_at = excluded.updated_at
	`

	_, err = db.conn.Exec(
		query,
		data.ID,
		data.URL,
		string(jsonData),
		data.FetchedAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	return nil
}

// GetByID retrieves scraped data by ID
func (db *DB) GetByID(id string) (*models.ScrapedData, error) {
	var jsonData string
	query := "SELECT data FROM scraped_data WHERE id = ?"

	err := db.conn.QueryRow(query, id).Scan(&jsonData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}

	var data models.ScrapedData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &data, nil
}

// GetByURL retrieves scraped data by URL
func (db *DB) GetByURL(url string) (*models.ScrapedData, error) {
	var jsonData string
	query := "SELECT data FROM scraped_data WHERE url = ?"

	err := db.conn.QueryRow(query, url).Scan(&jsonData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}

	var data models.ScrapedData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &data, nil
}

// DeleteByID deletes scraped data by ID
func (db *DB) DeleteByID(id string) error {
	result, err := db.conn.Exec("DELETE FROM scraped_data WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("no data found with id: %s", id)
	}

	return nil
}

// List returns all scraped data with optional pagination
func (db *DB) List(limit, offset int) ([]*models.ScrapedData, error) {
	query := `
		SELECT data FROM scraped_data
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	var results []*models.ScrapedData
	for rows.Next() {
		var jsonData string
		if err := rows.Scan(&jsonData); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var data models.ScrapedData
		if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		results = append(results, &data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// Count returns the total count of scraped data entries
func (db *DB) Count() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM scraped_data").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count data: %w", err)
	}
	return count, nil
}

// URLExists checks if a URL already exists in the database
func (db *DB) URLExists(url string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM scraped_data WHERE url = ?)"
	err := db.conn.QueryRow(query, url).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check URL existence: %w", err)
	}
	return exists, nil
}
