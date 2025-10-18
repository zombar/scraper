package ollama

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zombar/scraper/models"
)

const (
	DefaultBaseURL = "http://localhost:11434"
	DefaultModel   = "llama3.2"
	DefaultTimeout = 120 * time.Second
)

// Client is a client for interacting with Ollama
type Client struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

// NewClient creates a new Ollama client
func NewClient(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		model: model,
	}
}

// Generate sends a text generation request to Ollama
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := models.OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp models.OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

// GenerateWithVision sends a vision request to Ollama with an image
func (c *Client) GenerateWithVision(ctx context.Context, prompt string, imageData []byte) (string, error) {
	// Base64 encode the image
	encodedImage := base64.StdEncoding.EncodeToString(imageData)

	reqBody := models.OllamaVisionRequest{
		Model:  c.model,
		Prompt: prompt,
		Images: []string{encodedImage},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp models.OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

// ExtractContent uses Ollama to extract meaningful content from HTML text
func (c *Client) ExtractContent(ctx context.Context, rawText string) (string, error) {
	prompt := fmt.Sprintf(`You are a content extraction assistant. Given the following text extracted from a webpage, identify and return ONLY the meaningful human-readable content. Remove advertisements, navigation menus, footers, cookie notices, social media widgets, and other non-essential elements.

Return only the main content that a human would want to read. Do not add any commentary or explanations.

Text:
%s

Extracted content:`, rawText)

	return c.Generate(ctx, prompt)
}

// AnalyzeImage uses Ollama vision to generate a summary and tags for an image
func (c *Client) AnalyzeImage(ctx context.Context, imageData []byte, altText string) (summary string, tags []string, err error) {
	prompt := `Analyze this image and provide:
1. A 4-5 sentence summary describing what you see
2. A list of 5-10 relevant tags for categorizing the image

Format your response as JSON with the following structure:
{
  "summary": "Your 4-5 sentence description here",
  "tags": ["tag1", "tag2", "tag3"]
}`

	if altText != "" {
		prompt += fmt.Sprintf("\n\nImage alt text (may provide context): %s", altText)
	}

	response, err := c.GenerateWithVision(ctx, prompt, imageData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze image: %w", err)
	}

	// Strip markdown code blocks if present
	response = stripMarkdownCodeBlocks(response)

	// Parse JSON response
	var result struct {
		Summary string   `json:"summary"`
		Tags    []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If JSON parsing fails, try to extract manually
		return response, []string{}, nil
	}

	return result.Summary, result.Tags, nil
}

// stripMarkdownCodeBlocks removes markdown code block wrappers from a string
// This handles cases like ```json\n{...}\n``` and returns just the {...} content
func stripMarkdownCodeBlocks(s string) string {
	// Trim whitespace
	s = string(bytes.TrimSpace([]byte(s)))

	// Check if string starts with markdown code block
	if len(s) > 3 && s[:3] == "```" {
		// Find the end of the opening ```[language] line
		lines := bytes.Split([]byte(s), []byte("\n"))
		if len(lines) > 2 {
			// Remove first line (```json or similar) and last line (```)
			result := bytes.Join(lines[1:len(lines)-1], []byte("\n"))
			return string(bytes.TrimSpace(result))
		}
	}

	return s
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
