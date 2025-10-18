package scraper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zombar/scraper/models"
)

func TestNew(t *testing.T) {
	config := DefaultConfig()
	s := New(config)

	if s == nil {
		t.Fatal("Expected scraper to be non-nil")
	}

	if s.httpClient == nil {
		t.Error("Expected httpClient to be non-nil")
	}

	if s.ollamaClient == nil {
		t.Error("Expected ollamaClient to be non-nil")
	}
}

func TestExtractLinks(t *testing.T) {
	// Create mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response for both ExtractContent and link filtering
		var req models.OllamaRequest
		json.NewDecoder(r.Body).Decode(&req)

		var response string
		// Check if it's a link filtering request
		if contains(req.Prompt, "link filtering") {
			response = `["https://example.com/article-1", "https://example.com/article-2"]`
		} else {
			response = "Extracted article content"
		}

		resp := models.OllamaResponse{
			Response: response,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ollamaServer.Close()

	// Create mock web server
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<h1>Test Article</h1>
	<p>This is test content.</p>
	<a href="https://example.com/article-1">Article 1</a>
	<a href="https://example.com/article-2">Article 2</a>
	<a href="https://example.com/privacy">Privacy</a>
	<a href="/relative">Relative Link</a>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:   10 * time.Second,
		OllamaBaseURL: ollamaServer.URL,
		OllamaModel:   "test-model",
	}
	s := New(config)

	ctx := context.Background()
	links, err := s.ExtractLinks(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("ExtractLinks failed: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("Expected 2 sanitized links, got %d", len(links))
	}

	expectedLinks := []string{
		"https://example.com/article-1",
		"https://example.com/article-2",
	}

	for i, link := range links {
		if link != expectedLinks[i] {
			t.Errorf("Link[%d] = %s, want %s", i, link, expectedLinks[i])
		}
	}
}

func TestExtractLinksInvalidURL(t *testing.T) {
	config := DefaultConfig()
	s := New(config)

	ctx := context.Background()

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "invalid scheme",
			url:  "ftp://example.com",
		},
		{
			name: "malformed URL",
			url:  "ht!tp://invalid",
		},
		{
			name: "empty URL",
			url:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.ExtractLinks(ctx, tt.url)
			if err == nil {
				t.Error("Expected error for invalid URL, got nil")
			}
		})
	}
}

func TestExtractLinksHTTPError(t *testing.T) {
	// Create mock web server that returns errors
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer webServer.Close()

	config := DefaultConfig()
	s := New(config)

	ctx := context.Background()
	_, err := s.ExtractLinks(ctx, webServer.URL)

	if err == nil {
		t.Error("Expected error for HTTP 404, got nil")
	}
}

func TestExtractLinksMalformedHTML(t *testing.T) {
	// Create mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.OllamaResponse{
			Response: `[]`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ollamaServer.Close()

	// Create mock web server with malformed HTML
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><head><title>Test</title><body><p>Unclosed tags<a href="test"`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:   10 * time.Second,
		OllamaBaseURL: ollamaServer.URL,
		OllamaModel:   "test-model",
	}
	s := New(config)

	ctx := context.Background()
	// Should not panic, should handle gracefully
	links, err := s.ExtractLinks(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("ExtractLinks should handle malformed HTML gracefully: %v", err)
	}

	// Should return empty or minimal links
	if links == nil {
		t.Error("Expected non-nil links slice")
	}
}

func TestExtractLinksContextCancellation(t *testing.T) {
	// Create mock web server with delay
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("<html><body>Test</body></html>"))
	}))
	defer webServer.Close()

	config := DefaultConfig()
	s := New(config)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := s.ExtractLinks(ctx, webServer.URL)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestExtractLinksSanitizationFallback(t *testing.T) {
	// Create mock Ollama server that fails
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollamaServer.Close()

	// Create mock web server
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<a href="https://example.com/link1">Link 1</a>
	<a href="https://example.com/link2">Link 2</a>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:   10 * time.Second,
		OllamaBaseURL: ollamaServer.URL,
		OllamaModel:   "test-model",
	}
	s := New(config)

	ctx := context.Background()
	links, err := s.ExtractLinks(ctx, webServer.URL)

	// Should return all raw links when Ollama fails (fallback behavior)
	if err != nil {
		t.Errorf("ExtractLinks should not return error on Ollama failure: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("Expected 2 raw links as fallback, got %d", len(links))
	}

	expectedLinks := []string{
		"https://example.com/link1",
		"https://example.com/link2",
	}

	for i, link := range links {
		if link != expectedLinks[i] {
			t.Errorf("Link[%d] = %s, want %s", i, link, expectedLinks[i])
		}
	}
}

func TestExtractLinksEmptyPage(t *testing.T) {
	// Create mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.OllamaResponse{
			Response: `[]`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ollamaServer.Close()

	// Create mock web server with no links
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html><html><head><title>Empty</title></head><body><p>No links here</p></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:   10 * time.Second,
		OllamaBaseURL: ollamaServer.URL,
		OllamaModel:   "test-model",
	}
	s := New(config)

	ctx := context.Background()
	links, err := s.ExtractLinks(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("ExtractLinks failed: %v", err)
	}

	if len(links) != 0 {
		t.Errorf("Expected 0 links from empty page, got %d", len(links))
	}
}

func TestImageProcessing(t *testing.T) {
	// Create mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model  string   `json:"model"`
			Prompt string   `json:"prompt"`
			Images []string `json:"images"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// Check if it's an image analysis request
		if len(req.Images) > 0 {
			resp := models.OllamaResponse{
				Response: `{"summary": "A test image showing a red square on white background", "tags": ["test", "red", "square", "geometric"]}`,
				Done:     true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			resp := models.OllamaResponse{
				Response: "Extracted content",
				Done:     true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer ollamaServer.Close()

	// Create mock image server
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a simple 1x1 red pixel PNG
		imageData := []byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
			0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
			0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d, 0xb4, 0x00, 0x00, 0x00,
			0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}))
	defer imageServer.Close()

	// Create mock web server with image
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head><title>Test Page with Images</title></head>
<body>
	<h1>Test</h1>
	<img src="` + imageServer.URL + `/test.png" alt="Test image">
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:         10 * time.Second,
		OllamaBaseURL:       ollamaServer.URL,
		OllamaModel:         "test-model",
		EnableImageAnalysis: true,
		MaxImageSizeBytes:   10 * 1024 * 1024,
		ImageTimeout:        5 * time.Second,
	}
	s := New(config)

	ctx := context.Background()
	data, err := s.Scrape(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("Scrape failed: %v", err)
	}

	if len(data.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(data.Images))
	}

	img := data.Images[0]

	if img.URL != imageServer.URL+"/test.png" {
		t.Errorf("Image URL = %s, want %s", img.URL, imageServer.URL+"/test.png")
	}

	if img.AltText != "Test image" {
		t.Errorf("Alt text = %s, want 'Test image'", img.AltText)
	}

	if img.Summary == "" {
		t.Error("Expected image summary to be populated")
	}

	if len(img.Tags) == 0 {
		t.Error("Expected image tags to be populated")
	}

	t.Logf("Image summary: %s", img.Summary)
	t.Logf("Image tags: %v", img.Tags)
}

func TestImageProcessingDisabled(t *testing.T) {
	// Create mock web server with image
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
	<img src="https://example.com/image.jpg" alt="Test">
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:         10 * time.Second,
		OllamaBaseURL:       "http://localhost:11434",
		OllamaModel:         "test-model",
		EnableImageAnalysis: false, // Disabled
		MaxImageSizeBytes:   10 * 1024 * 1024,
		ImageTimeout:        5 * time.Second,
	}
	s := New(config)

	ctx := context.Background()
	data, err := s.Scrape(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("Scrape failed: %v", err)
	}

	if len(data.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(data.Images))
	}

	img := data.Images[0]

	// When disabled, summary and tags should be empty
	if img.Summary != "" {
		t.Errorf("Expected empty summary when image analysis disabled, got: %s", img.Summary)
	}

	if len(img.Tags) != 0 {
		t.Errorf("Expected empty tags when image analysis disabled, got: %v", img.Tags)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
