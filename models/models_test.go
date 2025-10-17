package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestScrapedDataJSONSerialization(t *testing.T) {
	now := time.Now()
	data := ScrapedData{
		URL:     "https://example.com",
		Title:   "Example Title",
		Content: "This is the content",
		Images: []ImageInfo{
			{
				URL:     "https://example.com/image.jpg",
				AltText: "An image",
				Summary: "This is a test image",
				Tags:    []string{"test", "example"},
			},
		},
		Links:          []string{"https://example.com/link1", "https://example.com/link2"},
		FetchedAt:      now,
		ProcessingTime: 1.5,
		Metadata: PageMetadata{
			Description:   "Test description",
			Keywords:      []string{"test", "example"},
			Author:        "Test Author",
			PublishedDate: "2024-01-01",
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal ScrapedData: %v", err)
	}

	// Unmarshal back
	var decoded ScrapedData
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ScrapedData: %v", err)
	}

	// Verify key fields
	if decoded.URL != data.URL {
		t.Errorf("URL mismatch: got %s, want %s", decoded.URL, data.URL)
	}
	if decoded.Title != data.Title {
		t.Errorf("Title mismatch: got %s, want %s", decoded.Title, data.Title)
	}
	if decoded.Content != data.Content {
		t.Errorf("Content mismatch: got %s, want %s", decoded.Content, data.Content)
	}
	if len(decoded.Images) != len(data.Images) {
		t.Errorf("Images length mismatch: got %d, want %d", len(decoded.Images), len(data.Images))
	}
	if len(decoded.Links) != len(data.Links) {
		t.Errorf("Links length mismatch: got %d, want %d", len(decoded.Links), len(data.Links))
	}
}

func TestImageInfoSerialization(t *testing.T) {
	img := ImageInfo{
		URL:     "https://example.com/image.jpg",
		AltText: "Test image",
		Summary: "A beautiful test image",
		Tags:    []string{"tag1", "tag2", "tag3"},
	}

	jsonData, err := json.Marshal(img)
	if err != nil {
		t.Fatalf("Failed to marshal ImageInfo: %v", err)
	}

	var decoded ImageInfo
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ImageInfo: %v", err)
	}

	if decoded.URL != img.URL {
		t.Errorf("URL mismatch: got %s, want %s", decoded.URL, img.URL)
	}
	if decoded.AltText != img.AltText {
		t.Errorf("AltText mismatch: got %s, want %s", decoded.AltText, img.AltText)
	}
	if decoded.Summary != img.Summary {
		t.Errorf("Summary mismatch: got %s, want %s", decoded.Summary, img.Summary)
	}
	if len(decoded.Tags) != len(img.Tags) {
		t.Errorf("Tags length mismatch: got %d, want %d", len(decoded.Tags), len(img.Tags))
	}
}

func TestOllamaRequestSerialization(t *testing.T) {
	req := OllamaRequest{
		Model:  "llama3.2",
		Prompt: "Test prompt",
		Stream: false,
		Format: "json",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal OllamaRequest: %v", err)
	}

	var decoded OllamaRequest
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal OllamaRequest: %v", err)
	}

	if decoded.Model != req.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, req.Model)
	}
	if decoded.Prompt != req.Prompt {
		t.Errorf("Prompt mismatch: got %s, want %s", decoded.Prompt, req.Prompt)
	}
	if decoded.Stream != req.Stream {
		t.Errorf("Stream mismatch: got %v, want %v", decoded.Stream, req.Stream)
	}
}

func TestPageMetadataOmitEmpty(t *testing.T) {
	// Test that empty fields are omitted from JSON
	metadata := PageMetadata{
		Description: "Test description",
		// Other fields empty
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal PageMetadata: %v", err)
	}

	jsonStr := string(jsonData)
	// Check that empty fields are not present
	if contains(jsonStr, "keywords") {
		t.Error("Expected keywords to be omitted from JSON")
	}
	if contains(jsonStr, "author") {
		t.Error("Expected author to be omitted from JSON")
	}
	if contains(jsonStr, "published_date") {
		t.Error("Expected published_date to be omitted from JSON")
	}
	if !contains(jsonStr, "description") {
		t.Error("Expected description to be present in JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
