# API Documentation

This document provides detailed information about the internal APIs and packages.

## Package: `models`

### Types

#### `ScrapedData`

The main output structure containing all scraped and processed data.

```go
type ScrapedData struct {
    URL            string        `json:"url"`
    Title          string        `json:"title"`
    Content        string        `json:"content"`
    Images         []ImageInfo   `json:"images"`
    Links          []string      `json:"links"`
    FetchedAt      time.Time     `json:"fetched_at"`
    ProcessingTime float64       `json:"processing_time_seconds"`
    Metadata       PageMetadata  `json:"metadata"`
}
```

**Fields:**
- `URL`: The scraped URL (may be modified to include scheme)
- `Title`: Page title extracted from `<title>` tag
- `Content`: AI-cleaned main content
- `Images`: Array of image information with AI analysis
- `Links`: Deduplicated list of all hyperlinks found
- `FetchedAt`: Timestamp when scraping started
- `ProcessingTime`: Total time in seconds for the entire operation
- `Metadata`: Additional page metadata from meta tags

#### `ImageInfo`

Information about an extracted image.

```go
type ImageInfo struct {
    URL        string   `json:"url"`
    AltText    string   `json:"alt_text"`
    Summary    string   `json:"summary"`
    Tags       []string `json:"tags"`
    Base64Data string   `json:"base64_data,omitempty"`
}
```

**Fields:**
- `URL`: Absolute URL of the image
- `AltText`: Alt text from the `<img>` tag
- `Summary`: 4-5 sentence AI-generated description
- `Tags`: AI-generated tags for categorization
- `Base64Data`: Base64 encoded image data (omitted from JSON if empty)

#### `PageMetadata`

Metadata extracted from HTML meta tags.

```go
type PageMetadata struct {
    Description   string   `json:"description,omitempty"`
    Keywords      []string `json:"keywords,omitempty"`
    Author        string   `json:"author,omitempty"`
    PublishedDate string   `json:"published_date,omitempty"`
}
```

All fields use `omitempty` and will be excluded from JSON if empty.

#### `OllamaRequest`

Request structure for Ollama API calls.

```go
type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
    Format string `json:"format,omitempty"`
}
```

#### `OllamaResponse`

Response structure from Ollama API.

```go
type OllamaResponse struct {
    Model     string `json:"model"`
    CreatedAt string `json:"created_at"`
    Response  string `json:"response"`
    Done      bool   `json:"done"`
}
```

#### `OllamaVisionRequest`

Vision request structure for image analysis.

```go
type OllamaVisionRequest struct {
    Model  string   `json:"model"`
    Prompt string   `json:"prompt"`
    Images []string `json:"images"` // base64 encoded
    Stream bool     `json:"stream"`
}
```

---

## Package: `ollama`

### Constants

```go
const (
    DefaultBaseURL = "http://localhost:11434"
    DefaultModel   = "llama3.2"
    DefaultTimeout = 120 * time.Second
)
```

### Types

#### `Client`

Client for interacting with Ollama.

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
    model      string
}
```

### Functions

#### `NewClient`

```go
func NewClient(baseURL, model string) *Client
```

Creates a new Ollama client. If `baseURL` or `model` are empty strings, defaults are used.

**Example:**
```go
client := ollama.NewClient("", "") // Uses defaults
client := ollama.NewClient("http://custom:11434", "llama3.1")
```

#### `Generate`

```go
func (c *Client) Generate(ctx context.Context, prompt string) (string, error)
```

Sends a text generation request to Ollama.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `prompt`: The prompt to send to the model

**Returns:**
- `string`: The generated response
- `error`: Any error that occurred

**Example:**
```go
response, err := client.Generate(ctx, "Summarize this text: ...")
if err != nil {
    log.Fatal(err)
}
```

#### `GenerateWithVision`

```go
func (c *Client) GenerateWithVision(ctx context.Context, prompt string, imageData []byte) (string, error)
```

Sends a vision request to Ollama with an image.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `prompt`: The prompt describing what to analyze
- `imageData`: Raw image bytes

**Returns:**
- `string`: The vision model's response
- `error`: Any error that occurred

#### `ExtractContent`

```go
func (c *Client) ExtractContent(ctx context.Context, rawText string) (string, error)
```

Uses Ollama to extract meaningful content from HTML text, removing ads and non-essential elements.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `rawText`: The raw text extracted from HTML

**Returns:**
- `string`: Cleaned content
- `error`: Any error that occurred

#### `AnalyzeImage`

```go
func (c *Client) AnalyzeImage(ctx context.Context, imageData []byte, altText string) (summary string, tags []string, err error)
```

Uses Ollama vision to generate a summary and tags for an image.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `imageData`: Raw image bytes
- `altText`: Optional alt text for context

**Returns:**
- `summary`: 4-5 sentence description
- `tags`: Array of relevant tags
- `err`: Any error that occurred

---

## Package: `scraper`

### Types

#### `Scraper`

Main scraper implementation.

```go
type Scraper struct {
    httpClient   *http.Client
    ollamaClient *ollama.Client
}
```

#### `Config`

Configuration for the scraper.

```go
type Config struct {
    HTTPTimeout   time.Duration
    OllamaBaseURL string
    OllamaModel   string
    UserAgent     string
}
```

**Fields:**
- `HTTPTimeout`: Timeout for HTTP requests
- `OllamaBaseURL`: Base URL for Ollama API
- `OllamaModel`: Model name to use
- `UserAgent`: User agent string for HTTP requests

### Functions

#### `DefaultConfig`

```go
func DefaultConfig() Config
```

Returns a default configuration.

**Defaults:**
- `HTTPTimeout`: 30 seconds
- `OllamaBaseURL`: http://localhost:11434
- `OllamaModel`: llama3.2
- `UserAgent`: Mozilla/5.0 (compatible; ContentScraper/1.0)

#### `New`

```go
func New(config Config) *Scraper
```

Creates a new Scraper instance with the given configuration.

**Example:**
```go
config := scraper.DefaultConfig()
config.HTTPTimeout = 60 * time.Second
s := scraper.New(config)
```

#### `Scrape`

```go
func (s *Scraper) Scrape(ctx context.Context, targetURL string) (*models.ScrapedData, error)
```

Scrapes a webpage and returns processed data.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `targetURL`: URL to scrape (scheme is optional, defaults to https)

**Returns:**
- `*models.ScrapedData`: Processed data
- `error`: Any error that occurred

**Example:**
```go
ctx := context.Background()
result, err := s.Scrape(ctx, "https://example.com")
if err != nil {
    log.Fatal(err)
}

jsonData, _ := json.MarshalIndent(result, "", "  ")
fmt.Println(string(jsonData))
```

**Processing Steps:**
1. Validates and normalizes URL
2. Fetches HTML content
3. Parses HTML structure
4. Extracts title, text, images, links, and metadata
5. Cleans content using Ollama
6. Analyzes each image with Ollama vision
7. Returns structured data with timing information

**Error Handling:**
- Returns error if URL is invalid
- Returns error if page fetch fails
- Returns error if HTML parsing fails
- Falls back to raw text if AI content cleaning fails
- Continues with other images if individual image processing fails

---

## Usage Examples

### Basic Scraping

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/zombar/scraper"
)

func main() {
    config := scraper.DefaultConfig()
    s := scraper.New(config)

    ctx := context.Background()
    result, err := s.Scrape(ctx, "https://example.com")
    if err != nil {
        log.Fatal(err)
    }

    jsonData, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(jsonData))
}
```

### Custom Configuration

```go
config := scraper.Config{
    HTTPTimeout:   60 * time.Second,
    OllamaBaseURL: "http://custom-host:11434",
    OllamaModel:   "llama3.1",
    UserAgent:     "MyBot/1.0",
}
s := scraper.New(config)
```

### With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

result, err := s.Scrape(ctx, "https://example.com")
```

### Direct Ollama Client Usage

```go
import "github.com/zombar/scraper/ollama"

client := ollama.NewClient("http://localhost:11434", "llama3.2")

// Text generation
response, err := client.Generate(ctx, "What is Go?")

// Content extraction
cleaned, err := client.ExtractContent(ctx, rawHTMLText)

// Image analysis
summary, tags, err := client.AnalyzeImage(ctx, imageBytes, "image alt text")
```

---

## Error Types

The application uses standard Go errors with descriptive messages. Common error patterns:

- `"invalid URL: ..."` - URL parsing failed
- `"failed to fetch page: ..."` - HTTP request failed
- `"failed to parse HTML: ..."` - HTML parsing failed
- `"ollama returned status XXX: ..."` - Ollama API error
- `"failed to analyze image: ..."` - Image processing failed

Always check errors and handle them appropriately:

```go
result, err := s.Scrape(ctx, url)
if err != nil {
    if strings.Contains(err.Error(), "invalid URL") {
        // Handle invalid URL
    } else if strings.Contains(err.Error(), "ollama") {
        // Handle Ollama errors
    } else {
        // Handle other errors
    }
}
```
