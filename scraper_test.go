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
