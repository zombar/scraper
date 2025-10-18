package scraper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestScoreLinkContent(t *testing.T) {
	// Create mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.OllamaResponse{
			Response: `{"score": 0.8, "reason": "High quality technical article", "categories": ["technical", "education"], "malicious_indicators": []}`,
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
	<title>Technical Article</title>
</head>
<body>
	<h1>Understanding Go Concurrency</h1>
	<p>This is a technical article about Go programming language concurrency patterns.</p>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:        10 * time.Second,
		OllamaBaseURL:      ollamaServer.URL,
		OllamaModel:        "test-model",
		LinkScoreThreshold: 0.5,
	}
	s := New(config)

	ctx := context.Background()
	score, err := s.ScoreLinkContent(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("ScoreLinkContent failed: %v", err)
	}

	if score.URL != webServer.URL {
		t.Errorf("URL = %s, want %s", score.URL, webServer.URL)
	}

	if score.Score != 0.8 {
		t.Errorf("Score = %f, want 0.8", score.Score)
	}

	if !score.IsRecommended {
		t.Error("Expected IsRecommended to be true for score 0.8 with threshold 0.5")
	}

	if score.Reason != "High quality technical article" {
		t.Errorf("Reason = %s, want 'High quality technical article'", score.Reason)
	}

	if len(score.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(score.Categories))
	}
}

func TestScoreLinkContentLowScore(t *testing.T) {
	// Create mock Ollama server returning low score
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.OllamaResponse{
			Response: `{"score": 0.2, "reason": "Social media platform", "categories": ["social_media"], "malicious_indicators": []}`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ollamaServer.Close()

	// Create mock web server
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html><html><head><title>Social Media</title></head><body><p>Social platform</p></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:        10 * time.Second,
		OllamaBaseURL:      ollamaServer.URL,
		OllamaModel:        "test-model",
		LinkScoreThreshold: 0.5,
	}
	s := New(config)

	ctx := context.Background()
	score, err := s.ScoreLinkContent(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("ScoreLinkContent failed: %v", err)
	}

	if score.Score != 0.2 {
		t.Errorf("Score = %f, want 0.2", score.Score)
	}

	if score.IsRecommended {
		t.Error("Expected IsRecommended to be false for score 0.2 with threshold 0.5")
	}

	if len(score.Categories) != 1 || score.Categories[0] != "social_media" {
		t.Errorf("Categories = %v, want ['social_media']", score.Categories)
	}
}

func TestScoreLinkContentMalicious(t *testing.T) {
	// Create mock Ollama server returning malicious indicators
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.OllamaResponse{
			Response: `{"score": 0.1, "reason": "Suspected phishing site", "categories": ["malicious"], "malicious_indicators": ["phishing", "suspicious_url"]}`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ollamaServer.Close()

	// Create mock web server
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html><html><head><title>Suspicious Site</title></head><body><p>Click here to win!</p></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:        10 * time.Second,
		OllamaBaseURL:      ollamaServer.URL,
		OllamaModel:        "test-model",
		LinkScoreThreshold: 0.5,
	}
	s := New(config)

	ctx := context.Background()
	score, err := s.ScoreLinkContent(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("ScoreLinkContent failed: %v", err)
	}

	if score.Score != 0.1 {
		t.Errorf("Score = %f, want 0.1", score.Score)
	}

	if score.IsRecommended {
		t.Error("Expected IsRecommended to be false for malicious content")
	}

	if len(score.MaliciousIndicators) != 2 {
		t.Errorf("Expected 2 malicious indicators, got %d", len(score.MaliciousIndicators))
	}
}

func TestScoreLinkContentInvalidURL(t *testing.T) {
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
			_, err := s.ScoreLinkContent(ctx, tt.url)
			if err == nil {
				t.Error("Expected error for invalid URL, got nil")
			}
		})
	}
}

func TestScoreLinkContentOllamaFailure(t *testing.T) {
	// Create mock Ollama server that fails
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ollamaServer.Close()

	// Create mock web server
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html><html><head><title>Test</title></head><body><p>Test content</p></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	config := Config{
		HTTPTimeout:        10 * time.Second,
		OllamaBaseURL:      ollamaServer.URL,
		OllamaModel:        "test-model",
		LinkScoreThreshold: 0.5,
	}
	s := New(config)

	ctx := context.Background()
	score, err := s.ScoreLinkContent(ctx, webServer.URL)

	// Should not error, should return default low score
	if err != nil {
		t.Fatalf("ScoreLinkContent should handle Ollama failure gracefully: %v", err)
	}

	if score.Score != 0.0 {
		t.Errorf("Expected score 0.0 on Ollama failure, got %f", score.Score)
	}

	if score.IsRecommended {
		t.Error("Expected IsRecommended to be false when Ollama fails")
	}
}

func TestScoreLinkContentCustomThreshold(t *testing.T) {
	// Create mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.OllamaResponse{
			Response: `{"score": 0.6, "reason": "Moderate quality content", "categories": ["business"], "malicious_indicators": []}`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ollamaServer.Close()

	// Create mock web server
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html><html><head><title>Business Article</title></head><body><p>Business content</p></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer webServer.Close()

	tests := []struct {
		name          string
		threshold     float64
		shouldBeRecommended bool
	}{
		{
			name:          "threshold 0.5",
			threshold:     0.5,
			shouldBeRecommended: true, // 0.6 >= 0.5
		},
		{
			name:          "threshold 0.7",
			threshold:     0.7,
			shouldBeRecommended: false, // 0.6 < 0.7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				HTTPTimeout:        10 * time.Second,
				OllamaBaseURL:      ollamaServer.URL,
				OllamaModel:        "test-model",
				LinkScoreThreshold: tt.threshold,
			}
			s := New(config)

			ctx := context.Background()
			score, err := s.ScoreLinkContent(ctx, webServer.URL)

			if err != nil {
				t.Fatalf("ScoreLinkContent failed: %v", err)
			}

			if score.IsRecommended != tt.shouldBeRecommended {
				t.Errorf("IsRecommended = %v, want %v (threshold %f, score %f)",
					score.IsRecommended, tt.shouldBeRecommended, tt.threshold, score.Score)
			}
		})
	}
}

func TestScrapeIncludesScore(t *testing.T) {
	// Create mock Ollama server that returns scoring
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return different responses based on the request
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		prompt, _ := reqBody["prompt"].(string)

		// Scoring request
		if containsHelper(prompt, "quality score") || containsHelper(prompt, "quality assessment") {
			resp := models.OllamaResponse{
				Response: `{"score": 0.85, "reason": "High quality technical content", "categories": ["technical", "education"], "malicious_indicators": []}`,
				Done:     true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Content extraction or link filtering - just return simple text
		resp := models.OllamaResponse{
			Response: "Cleaned content",
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
	<title>Test Article</title>
	<meta name="description" content="Test description">
</head>
<body>
	<h1>Test Content</h1>
	<p>This is test content for scraping.</p>
	<a href="/link1">Link 1</a>
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
		LinkScoreThreshold:  0.5,
		EnableImageAnalysis: false, // Disable to simplify test
	}
	s := New(config)

	ctx := context.Background()
	data, err := s.Scrape(ctx, webServer.URL)

	if err != nil {
		t.Fatalf("Scrape failed: %v", err)
	}

	// Verify basic scraped data
	if data.URL != webServer.URL {
		t.Errorf("URL = %s, want %s", data.URL, webServer.URL)
	}

	if data.Title == "" {
		t.Error("Expected non-empty title")
	}

	// Verify score metadata is present
	if data.Score == nil {
		t.Fatal("Expected Score to be present in ScrapedData")
	}

	if data.Score.Score != 0.85 {
		t.Errorf("Score = %f, want 0.85", data.Score.Score)
	}

	if data.Score.Reason != "High quality technical content" {
		t.Errorf("Reason = %s, want 'High quality technical content'", data.Score.Reason)
	}

	if len(data.Score.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(data.Score.Categories))
	}

	if !data.Score.IsRecommended {
		t.Error("Expected IsRecommended to be true for score 0.85 with threshold 0.5")
	}

	t.Logf("✓ Scrape includes score metadata: score=%.2f, recommended=%v",
		data.Score.Score, data.Score.IsRecommended)
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

// TestScoreContentFallbackSocialMedia tests fallback scoring for social media
func TestScoreContentFallbackSocialMedia(t *testing.T) {
	score, reason, categories, indicators := scoreContentFallback(
		"https://www.facebook.com/profile",
		"Facebook Profile",
		"This is my Facebook profile with posts and photos.",
	)

	if score != 0.1 {
		t.Errorf("Expected score 0.1 for social media, got %.2f", score)
	}

	if !containsString(categories, "social_media") {
		t.Error("Expected 'social_media' category")
	}

	if !containsString(categories, "low_quality") {
		t.Error("Expected 'low_quality' category")
	}

	if !strings.Contains(reason, "Blocked content type") {
		t.Errorf("Expected reason to mention blocked content, got: %s", reason)
	}

	if len(indicators) == 0 {
		t.Error("Expected malicious indicators for social media")
	}
}

// TestScoreContentFallbackQualityDomain tests fallback scoring for quality domains
func TestScoreContentFallbackQualityDomain(t *testing.T) {
	score, reason, categories, _ := scoreContentFallback(
		"https://en.wikipedia.org/wiki/Artificial_Intelligence",
		"Artificial Intelligence - Wikipedia",
		strings.Repeat("This is a comprehensive article about artificial intelligence. ", 50),
	)

	if score < 0.7 {
		t.Errorf("Expected high score for Wikipedia, got %.2f", score)
	}

	if !containsString(categories, "reference") || !containsString(categories, "trusted_source") {
		t.Errorf("Expected quality categories, got: %v", categories)
	}

	if !strings.Contains(reason, "Quality domain") {
		t.Errorf("Expected reason to mention quality domain, got: %s", reason)
	}
}

// TestScoreContentFallbackShortContent tests fallback scoring for short content
func TestScoreContentFallbackShortContent(t *testing.T) {
	score, reason, categories, _ := scoreContentFallback(
		"https://example.com/short",
		"Short Page",
		"Very short content here.",
	)

	if score >= 0.5 {
		t.Errorf("Expected low score for short content, got %.2f", score)
	}

	if !containsString(categories, "low_quality") {
		t.Errorf("Expected 'low_quality' category, got: %v", categories)
	}

	if !strings.Contains(reason, "short") {
		t.Errorf("Expected reason to mention short content, got: %s", reason)
	}
}

// TestScoreContentFallbackSpam tests fallback scoring for spam content
func TestScoreContentFallbackSpam(t *testing.T) {
	spamContent := "Click here! Click here! Click here! Buy now! Buy now! Limited offer!"
	score, reason, categories, indicators := scoreContentFallback(
		"https://example.com/spam",
		"Amazing Offer",
		spamContent,
	)

	if score >= 0.3 {
		t.Errorf("Expected very low score for spam, got %.2f", score)
	}

	if !containsString(categories, "spam") {
		t.Errorf("Expected 'spam' category, got: %v", categories)
	}

	if !strings.Contains(reason, "Spam indicators") {
		t.Errorf("Expected reason to mention spam, got: %s", reason)
	}

	if !containsString(indicators, "spam_keywords") {
		t.Errorf("Expected spam_keywords in malicious indicators, got: %v", indicators)
	}
}

// TestScoreContentFallbackTechnical tests fallback scoring for technical content
func TestScoreContentFallbackTechnical(t *testing.T) {
	technicalContent := strings.Repeat("This is a technical guide about software development and programming best practices. ", 20)
	score, reason, categories, _ := scoreContentFallback(
		"https://example.com/tutorial",
		"Software Development Tutorial",
		technicalContent,
	)

	if score < 0.6 {
		t.Errorf("Expected good score for technical content, got %.2f", score)
	}

	if !containsString(categories, "technical") || !containsString(categories, "educational") {
		t.Errorf("Expected technical/educational categories, got: %v", categories)
	}

	if !strings.Contains(reason, "Rule-based") {
		t.Errorf("Expected reason to mention rule-based assessment, got: %s", reason)
	}
}

// TestScoreContentFallbackGambling tests fallback scoring for gambling sites
func TestScoreContentFallbackGambling(t *testing.T) {
	score, _, categories, indicators := scoreContentFallback(
		"https://www.betcasino.com",
		"Online Casino",
		"Place your bets and win big!",
	)

	if score != 0.1 {
		t.Errorf("Expected score 0.1 for gambling site, got %.2f", score)
	}

	if !containsString(categories, "gambling") {
		t.Errorf("Expected 'gambling' category, got: %v", categories)
	}

	if len(indicators) == 0 {
		t.Error("Expected malicious indicators for gambling site")
	}
}

// TestScrapeWithFallbackScoring tests that scraping works with fallback scoring when Ollama is down
func TestScrapeWithFallbackScoring(t *testing.T) {
	// Create a mock web server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := `<html><head><title>Test Article</title></head><body>` +
			strings.Repeat("<p>This is a substantial article about important topics. </p>", 30) +
			`</body></html>`
		w.Write([]byte(html))
	})
	webServer := httptest.NewServer(handler)
	defer webServer.Close()

	// Create scraper WITHOUT Ollama client (will fail and use fallback)
	config := DefaultConfig()
	config.LinkScoreThreshold = 0.5
	s := New(config)

	ctx := context.Background()
	data, err := s.Scrape(ctx, webServer.URL)
	if err != nil {
		t.Fatalf("Scrape failed: %v", err)
	}

	// Verify score is present (from fallback)
	if data.Score == nil {
		t.Fatal("Expected Score to be present from fallback scoring")
	}

	// Score should be decent for substantial content
	if data.Score.Score < 0.4 {
		t.Errorf("Expected reasonable fallback score for good content, got %.2f", data.Score.Score)
	}

	// Reason should indicate rule-based assessment
	if !strings.Contains(data.Score.Reason, "Rule-based") {
		t.Errorf("Expected reason to indicate rule-based fallback, got: %s", data.Score.Reason)
	}

	// Categories should not be empty
	if len(data.Score.Categories) == 0 {
		t.Error("Expected categories from fallback scoring")
	}

	// Verify AIUsed is false for rule-based fallback
	if data.Score.AIUsed {
		t.Error("Expected AIUsed to be false for rule-based fallback")
	}
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
