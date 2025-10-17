package models

import "time"

// ScrapedData represents the complete output of a web scraping operation
type ScrapedData struct {
	URL            string       `json:"url"`
	Title          string       `json:"title"`
	Content        string       `json:"content"`
	Images         []ImageInfo  `json:"images"`
	Links          []string     `json:"links"`
	FetchedAt      time.Time    `json:"fetched_at"`
	ProcessingTime float64      `json:"processing_time_seconds"`
	Metadata       PageMetadata `json:"metadata"`
}

// ImageInfo contains information about an extracted image
type ImageInfo struct {
	URL     string   `json:"url"`
	AltText string   `json:"alt_text"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

// PageMetadata contains additional metadata about the scraped page
type PageMetadata struct {
	Description   string   `json:"description,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	Author        string   `json:"author,omitempty"`
	PublishedDate string   `json:"published_date,omitempty"`
}

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// OllamaVisionRequest represents a vision request to the Ollama API
type OllamaVisionRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"` // base64 encoded images
	Stream bool     `json:"stream"`
}
