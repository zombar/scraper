# Web Scraper with AI Content Processing

A Go-based web scraping tool that leverages a local Ollama instance to intelligently extract and process web content. Available as both a CLI tool and a REST API server with persistent storage. This tool fetches web pages, extracts meaningful content, analyzes images, and provides structured JSON output.

## Features

### Core Scraping
- **Intelligent Content Extraction**: Uses AI to identify and extract meaningful human-readable content while filtering out advertisements, navigation menus, and other non-essential elements
- **Image Analysis**: Processes images with AI vision models to generate summaries and tags
- **Link Extraction**: Collects all relevant hyperlinks from pages
- **Metadata Collection**: Extracts page metadata including title, description, keywords, author, and publication date
- **Performance Metrics**: Tracks fetch and processing time for each operation
- **JSON Output**: Returns all data in a structured JSON format

### API Server (New!)
- **REST API**: Full-featured HTTP API for web integration
- **Persistent Storage**: SQLite database with automatic migrations (PostgreSQL-ready)
- **Batch Processing**: Process multiple URLs concurrently (up to 50 per request)
- **Smart Caching**: Automatically returns cached results for previously scraped URLs
- **UUID-based IDs**: Each result gets a unique identifier for retrieval and deletion
- **CORS Support**: Ready for web frontend integration
- **Graceful Shutdown**: Handles interrupts cleanly

## Requirements

- Go 1.24 or higher
- [Ollama](https://ollama.ai) running locally
- Required Ollama models:
  - `llama3.2` (or your preferred text model)
  - `llama3.2-vision` (for image analysis)

### Installing Ollama Models

```bash
# Install the text model
ollama pull llama3.2

# Install the vision model for image analysis
ollama pull llama3.2-vision
```

## Installation

1. Clone or download this repository
2. Build the application:

```bash
# Build CLI tool
go build -o scraper-bin

# Build API server
go build -o scraper-api ./cmd/api

# Or using Make
make build        # CLI only
make build-api    # API server only
make build-both   # Both CLI and API
```

## Quick Start

### CLI Tool

```bash
# Using Go directly
./scraper-bin -url "https://example.com" -pretty

# Using Make
make run URL=https://example.com
```

### API Server

```bash
# Start API server (default port 8080)
./scraper-api

# Or with custom settings
./scraper-api -addr :3000 -db ./data/scraper.db

# Using Make
make run-api
make run-api PORT=3000 DB=./data/scraper.db
```

Then use the API:
```bash
# Scrape a URL
curl -X POST http://localhost:8080/api/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# Get health status
curl http://localhost:8080/health
```

See [API_SERVER.md](API_SERVER.md) for complete API documentation.

## Make Commands

This project includes a Makefile with convenient shortcuts:

```bash
# Show all available commands
make help

# Build commands
make build        # Build CLI application
make build-api    # Build API server
make build-both   # Build both

# Run commands
make run URL=https://example.com              # Run CLI
make run-api                                  # Run API server
make run-api PORT=3000 DB=./data/scraper.db  # Run API with custom settings

# Testing
make test           # Run all tests
make test-coverage  # Run with coverage report
make coverage-html  # Generate HTML coverage

# Development
make clean  # Clean build artifacts
make check  # Format, vet, and test
make fmt    # Format code
make vet    # Run go vet

# Cross-compile
make build-cross  # Build for multiple platforms
```

## Usage

### Basic Usage

```bash
./scraper-bin -url "https://example.com"

# Or with Make
make run URL=https://example.com
```

### Command-Line Options

- `-url` (required): The URL to scrape
- `-timeout`: Request timeout duration (default: 120s)
- `-ollama-url`: Ollama base URL (default: http://localhost:11434)
- `-ollama-model`: Ollama model to use (default: llama3.2)
- `-pretty`: Pretty print JSON output (default: false)

### Examples

```bash
# Basic scraping with pretty output
./scraper-bin -url "https://example.com/article" -pretty

# Custom timeout and Ollama configuration
./scraper-bin -url "https://example.com" -timeout 60s -ollama-url "http://custom-host:11434"

# Using a different model
./scraper-bin -url "https://example.com" -ollama-model "llama3.1"

# Save output to file
./scraper-bin -url "https://example.com" -pretty > output.json

# Using Make for common tasks
make run URL=https://example.com  # Basic run with pretty output
make test-coverage                 # Run tests with coverage
make check                         # Format, vet, and test
```

## Output Format

The tool returns a JSON object with the following structure:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com",
  "title": "Page Title",
  "content": "The main content extracted and cleaned by AI...",
  "images": [
    {
      "url": "https://example.com/image.jpg",
      "alt_text": "Image description",
      "summary": "AI-generated 4-5 sentence description of the image",
      "tags": ["tag1", "tag2", "tag3"]
    }
  ],
  "links": [
    "https://example.com/page1",
    "https://example.com/page2"
  ],
  "fetched_at": "2024-01-01T12:00:00Z",
  "created_at": "2024-01-01T12:00:00Z",
  "processing_time_seconds": 3.45,
  "cached": false,
  "metadata": {
    "description": "Page meta description",
    "keywords": ["keyword1", "keyword2"],
    "author": "Author Name",
    "published_date": "2024-01-01"
  }
}
```

**Key Fields:**
- `id`: Unique UUID identifier
- `cached`: `true` if served from cache, `false` if freshly scraped
- `created_at`: When the data was originally created (useful for detecting stale cache)
- `fetched_at`: Same as `created_at` for fresh scrapes

## Architecture

The application is structured into several packages:

### `models/`
Defines data structures for the application:
- `ScrapedData`: Main output structure
- `ImageInfo`: Image information with AI analysis
- `PageMetadata`: Page metadata from HTML meta tags
- `OllamaRequest/Response`: Ollama API communication

### `ollama/`
Ollama client implementation:
- `Client`: HTTP client for Ollama API
- `Generate()`: Text generation for content extraction
- `GenerateWithVision()`: Image analysis with vision models
- `ExtractContent()`: AI-powered content cleaning
- `AnalyzeImage()`: Image summarization and tagging

### `scraper/`
Core scraping functionality:
- `Scraper`: Main scraper implementation
- HTML parsing and content extraction
- Image and link discovery
- Metadata extraction
- Integration with Ollama for AI processing

## Development

### Make Commands

The Makefile provides convenient shortcuts for common development tasks:

```bash
make help           # Show all available commands
make build          # Build the application
make test           # Run all tests
make test-verbose   # Run tests with verbose output
make test-coverage  # Run tests with coverage
make coverage-html  # Generate HTML coverage report
make clean          # Remove build artifacts
make fmt            # Format code
make vet            # Run go vet
make check          # Run fmt, vet, and test
make all            # Run check and build
make build-all      # Build for multiple platforms
```

### Running Tests

The project includes a comprehensive test suite:

```bash
# Using Make (recommended)
make test
make test-coverage
make coverage-html

# Using Go directly
go test ./...
go test -cover ./...
go test -v ./...
go test ./scraper
```

### Project Structure

```
.
├── main.go              # CLI entry point
├── Makefile             # Build automation and shortcuts
├── models/              # Data structures
│   ├── models.go
│   └── models_test.go
├── ollama/              # Ollama client
│   ├── client.go
│   └── client_test.go
├── scraper/             # Core scraping logic
│   ├── scraper.go
│   └── scraper_test.go
├── go.mod               # Go module definition
├── README.md            # User documentation
├── API.md               # API reference
└── DEVELOPMENT.md       # Developer guide
```

## How It Works

1. **Fetch**: Downloads the HTML content from the target URL
2. **Parse**: Parses HTML using Go's `html` package
3. **Extract**: Extracts raw text, images, links, and metadata
4. **Clean**: Sends raw text to Ollama to remove ads and non-content elements
5. **Analyze**: Processes each image with Ollama vision to generate summaries and tags
6. **Format**: Structures all data into JSON format with timing metrics

## Limitations

- The application does not persist data - all output is returned as JSON
- JavaScript-rendered content is not supported (no headless browser)
- Large pages with many images may take significant time to process
- Image analysis requires downloading each image, which can be slow
- Requires a running Ollama instance with appropriate models

## Error Handling

The application handles various error conditions:
- Invalid URLs
- Network timeouts
- HTTP errors (404, 500, etc.)
- Ollama connection issues
- Malformed HTML
- Image download failures

When errors occur during image processing, the tool continues processing other images rather than failing completely. If AI content extraction fails, the tool falls back to raw text extraction.

## Performance Considerations

- **Timeouts**: Default timeout is 120s but can be adjusted for large pages
- **Ollama Performance**: Processing time depends on your Ollama installation and available hardware
- **Image Processing**: Each image requires a separate Ollama vision API call
- **Concurrent Processing**: Currently processes images sequentially (could be parallelized in future versions)

## Contributing

This is a backend-only application focused on data extraction. When contributing:
- Maintain test coverage for new features
- Follow Go best practices and idioms
- Prefer standard library over external dependencies
- Keep the application stateless

## License

This project is provided as-is for educational and development purposes.
