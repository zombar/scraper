package db

import (
	"os"
	"testing"
	"time"

	"github.com/zombar/scraper/models"
)

func setupTestDB(t *testing.T) *DB {
	// Use in-memory database for tests
	config := Config{
		Driver: "sqlite",
		DSN:    ":memory:",
	}

	db, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

func TestNewDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if db.conn == nil {
		t.Error("Database connection is nil")
	}
}

func TestMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check that tables were created
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM scraped_data").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query scraped_data table: %v", err)
	}

	err = db.conn.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query schema_migrations table: %v", err)
	}
}

func TestSaveAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	data := &models.ScrapedData{
		ID:             "test-id-123",
		URL:            "https://example.com",
		Title:          "Test Page",
		Content:        "Test content",
		Images:         []models.ImageInfo{},
		Links:          []string{"https://example.com/link1"},
		FetchedAt:      time.Now(),
		ProcessingTime: 1.5,
		Metadata: models.PageMetadata{
			Description: "Test description",
		},
	}

	// Save data
	err := db.SaveScrapedData(data)
	if err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	// Retrieve data
	retrieved, err := db.GetByID("test-id-123")
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved data is nil")
	}

	if retrieved.ID != data.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, data.ID)
	}

	if retrieved.URL != data.URL {
		t.Errorf("URL mismatch: got %s, want %s", retrieved.URL, data.URL)
	}

	if retrieved.Title != data.Title {
		t.Errorf("Title mismatch: got %s, want %s", retrieved.Title, data.Title)
	}
}

func TestGetByURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	data := &models.ScrapedData{
		ID:             "test-id-456",
		URL:            "https://example.com/test",
		Title:          "Test Page",
		Content:        "Test content",
		Images:         []models.ImageInfo{},
		Links:          []string{},
		FetchedAt:      time.Now(),
		ProcessingTime: 1.0,
	}

	err := db.SaveScrapedData(data)
	if err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	retrieved, err := db.GetByURL("https://example.com/test")
	if err != nil {
		t.Fatalf("Failed to get data by URL: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved data is nil")
	}

	if retrieved.URL != data.URL {
		t.Errorf("URL mismatch: got %s, want %s", retrieved.URL, data.URL)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	retrieved, err := db.GetByID("nonexistent-id")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil for nonexistent ID")
	}
}

func TestDeleteByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	data := &models.ScrapedData{
		ID:             "delete-test",
		URL:            "https://example.com/delete",
		Title:          "Delete Test",
		Content:        "Content",
		FetchedAt:      time.Now(),
		ProcessingTime: 1.0,
	}

	// Save data
	err := db.SaveScrapedData(data)
	if err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	// Delete data
	err = db.DeleteByID("delete-test")
	if err != nil {
		t.Fatalf("Failed to delete data: %v", err)
	}

	// Verify deletion
	retrieved, err := db.GetByID("delete-test")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	if retrieved != nil {
		t.Error("Data was not deleted")
	}
}

func TestDeleteByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.DeleteByID("nonexistent-id")
	if err == nil {
		t.Error("Expected error when deleting nonexistent ID")
	}
}

func TestList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Save multiple entries
	for i := 0; i < 5; i++ {
		data := &models.ScrapedData{
			ID:             string(rune('a' + i)),
			URL:            "https://example.com/" + string(rune('a'+i)),
			Title:          "Test " + string(rune('a'+i)),
			Content:        "Content",
			FetchedAt:      time.Now(),
			ProcessingTime: 1.0,
		}
		if err := db.SaveScrapedData(data); err != nil {
			t.Fatalf("Failed to save data: %v", err)
		}
	}

	// List with limit
	results, err := db.List(3, 0)
	if err != nil {
		t.Fatalf("Failed to list data: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// List with offset
	results, err = db.List(10, 2)
	if err != nil {
		t.Fatalf("Failed to list data with offset: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results with offset, got %d", len(results))
	}
}

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initial count should be 0
	count, err := db.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	// Add entries
	for i := 0; i < 3; i++ {
		data := &models.ScrapedData{
			ID:             string(rune('a' + i)),
			URL:            "https://example.com/" + string(rune('a'+i)),
			Title:          "Test",
			Content:        "Content",
			FetchedAt:      time.Now(),
			ProcessingTime: 1.0,
		}
		if err := db.SaveScrapedData(data); err != nil {
			t.Fatalf("Failed to save data: %v", err)
		}
	}

	// Count should be 3
	count, err = db.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestURLExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	url := "https://example.com/exists"

	// Check non-existent URL
	exists, err := db.URLExists(url)
	if err != nil {
		t.Fatalf("URLExists returned error: %v", err)
	}

	if exists {
		t.Error("URL should not exist initially")
	}

	// Add data
	data := &models.ScrapedData{
		ID:             "exists-test",
		URL:            url,
		Title:          "Test",
		Content:        "Content",
		FetchedAt:      time.Now(),
		ProcessingTime: 1.0,
	}

	if err := db.SaveScrapedData(data); err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	// Check existing URL
	exists, err = db.URLExists(url)
	if err != nil {
		t.Fatalf("URLExists returned error: %v", err)
	}

	if !exists {
		t.Error("URL should exist after saving")
	}
}

func TestUpsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	url := "https://example.com/upsert"

	// Initial save
	data1 := &models.ScrapedData{
		ID:             "upsert-1",
		URL:            url,
		Title:          "Original Title",
		Content:        "Original content",
		FetchedAt:      time.Now(),
		ProcessingTime: 1.0,
	}

	if err := db.SaveScrapedData(data1); err != nil {
		t.Fatalf("Failed to save initial data: %v", err)
	}

	// Update with same URL, different ID
	data2 := &models.ScrapedData{
		ID:             "upsert-2",
		URL:            url,
		Title:          "Updated Title",
		Content:        "Updated content",
		FetchedAt:      time.Now(),
		ProcessingTime: 2.0,
	}

	if err := db.SaveScrapedData(data2); err != nil {
		t.Fatalf("Failed to update data: %v", err)
	}

	// Retrieve and verify it was updated
	retrieved, err := db.GetByURL(url)
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}

	if retrieved.ID != "upsert-2" {
		t.Errorf("Expected ID 'upsert-2', got %s", retrieved.ID)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", retrieved.Title)
	}

	// Verify old ID doesn't exist
	old, err := db.GetByID("upsert-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	if old != nil {
		t.Error("Old ID should not exist after upsert")
	}
}

func TestFileDatabase(t *testing.T) {
	// Test with actual file database
	dbPath := "test-scraper.db"
	defer os.Remove(dbPath)

	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}

	db, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create file database: %v", err)
	}

	// Save data
	data := &models.ScrapedData{
		ID:             "file-test",
		URL:            "https://example.com/file",
		Title:          "File Test",
		Content:        "Content",
		FetchedAt:      time.Now(),
		ProcessingTime: 1.0,
	}

	if err := db.SaveScrapedData(data); err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	db.Close()

	// Reopen and verify data persisted
	db, err = New(config)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	retrieved, err := db.GetByID("file-test")
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Data was not persisted")
	}

	if retrieved.Title != "File Test" {
		t.Errorf("Title mismatch: got %s, want File Test", retrieved.Title)
	}
}
