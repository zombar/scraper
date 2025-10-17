# Development Guide

This guide provides information for developers working on the web scraper project.

## Prerequisites

- Go 1.24 or higher
- Ollama installed and running locally
- Basic understanding of Go, HTTP, and HTML parsing

## Getting Started

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Ensure Ollama is running:
   ```bash
   ollama serve
   ```
4. Run tests to verify setup:
   ```bash
   go test ./...
   ```

## Project Structure

```
.
├── main.go              # CLI entry point
├── models/              # Data structures and types
├── ollama/              # Ollama API client
├── scraper/             # Core scraping logic
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── README.md            # User documentation
├── API.md               # API reference
└── DEVELOPMENT.md       # This file
```

## Development Workflow

### 1. Making Changes

- Create feature branches from `main`
- Write tests for new functionality
- Ensure all tests pass before committing
- Follow Go conventions and best practices

### 2. Testing

Run the full test suite:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Run tests for a specific package:

```bash
go test ./scraper -v
go test ./ollama -v
go test ./models -v
```

### 3. Building

Build the application:

```bash
go build -o scraper
```

Build with optimization:

```bash
go build -ldflags="-s -w" -o scraper
```

### 4. Code Quality

Format code:

```bash
go fmt ./...
```

Run linter:

```bash
go vet ./...
```

Check for common mistakes:

```bash
staticcheck ./...
```

## Package Guidelines

### `models` Package

- Define all data structures here
- Use JSON tags for serialization
- Use `omitempty` for optional fields
- Keep types simple and focused
- Add validation methods if needed

Example:

```go
type NewType struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2,omitempty"`
}
```

### `ollama` Package

- All Ollama API interactions go here
- Use context for cancellation and timeouts
- Return descriptive errors
- Test with mock HTTP servers

Example test pattern:

```go
func TestNewFeature(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock Ollama response
        resp := models.OllamaResponse{
            Response: "test response",
            Done:     true,
        }
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    client := ollama.NewClient(server.URL, "test-model")
    // Test the feature...
}
```

### `scraper` Package

- Core scraping logic lives here
- Parse HTML using `golang.org/x/net/html`
- Handle errors gracefully
- Resolve relative URLs to absolute URLs
- Filter out non-content elements

Example HTML traversal:

```go
func extractSomething(n *html.Node) []string {
    var results []string
    var traverse func(*html.Node)

    traverse = func(node *html.Node) {
        if node.Type == html.ElementNode && node.Data == "targetTag" {
            // Extract data
        }
        for c := node.FirstChild; c != nil; c = c.NextSibling {
            traverse(c)
        }
    }
    traverse(n)
    return results
}
```

## Testing Guidelines

### Unit Tests

- Test each function in isolation
- Use table-driven tests for multiple cases
- Mock external dependencies (HTTP, Ollama)
- Aim for high coverage (>80%)

Example table-driven test:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"case1", "input1", "output1", false},
        {"case2", "input2", "output2", false},
        {"error case", "bad", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Function() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Function() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

- Test full workflows
- Use mock HTTP servers for both web and Ollama
- Verify complete data structures
- Test error handling paths

### Test Naming Conventions

- `Test<FunctionName>` for unit tests
- `Test<FunctionName><Condition>` for specific cases
- `Test<Package>Integration` for integration tests

## Adding New Features

### Example: Adding a New Extraction Feature

1. **Define the data structure** in `models/models.go`:

```go
type NewFeature struct {
    Field1 string `json:"field1"`
    Field2 string `json:"field2"`
}
```

2. **Add to ScrapedData**:

```go
type ScrapedData struct {
    // ... existing fields ...
    NewFeature NewFeature `json:"new_feature"`
}
```

3. **Implement extraction** in `scraper/scraper.go`:

```go
func (s *Scraper) extractNewFeature(n *html.Node) NewFeature {
    // Implementation
}
```

4. **Integrate into Scrape()**:

```go
func (s *Scraper) Scrape(ctx context.Context, targetURL string) (*models.ScrapedData, error) {
    // ... existing code ...
    newFeature := s.extractNewFeature(doc)

    return &models.ScrapedData{
        // ... existing fields ...
        NewFeature: newFeature,
    }, nil
}
```

5. **Write tests** in `scraper/scraper_test.go`:

```go
func TestExtractNewFeature(t *testing.T) {
    // Test implementation
}
```

6. **Update documentation** in README.md and API.md

## Debugging

### Enable Verbose HTTP Logging

```go
import "net/http/httputil"

// In scraper.go
resp, err := s.httpClient.Do(req)
dump, _ := httputil.DumpResponse(resp, true)
fmt.Printf("Response:\n%s\n", dump)
```

### Debug HTML Parsing

```go
import "golang.org/x/net/html"

// Print HTML tree structure
func printNode(n *html.Node, depth int) {
    if n.Type == html.ElementNode {
        fmt.Printf("%s<%s>\n", strings.Repeat("  ", depth), n.Data)
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        printNode(c, depth+1)
    }
}
```

### Test with Local HTML File

```go
func TestWithLocalFile(t *testing.T) {
    content, _ := os.ReadFile("testdata/sample.html")
    doc, _ := html.Parse(strings.NewReader(string(content)))
    // Test with doc
}
```

## Performance Optimization

### Current Performance Characteristics

- HTML parsing: Very fast (< 100ms for typical pages)
- Text extraction: Fast (< 50ms)
- AI content cleaning: Depends on Ollama (1-5 seconds)
- Image analysis: Slow (2-10 seconds per image)

### Optimization Opportunities

1. **Parallel Image Processing**:
   ```go
   var wg sync.WaitGroup
   imageChan := make(chan models.ImageInfo, len(imageURLs))

   for _, imgURL := range imageURLs {
       wg.Add(1)
       go func(url ImageURL) {
           defer wg.Done()
           imgInfo, err := s.processImage(ctx, url)
           if err == nil {
               imageChan <- imgInfo
           }
       }(imgURL)
   }

   go func() {
       wg.Wait()
       close(imageChan)
   }()

   for img := range imageChan {
       images = append(images, img)
   }
   ```

2. **Caching**: Consider adding optional caching for:
   - Ollama responses
   - Downloaded images
   - Parsed HTML

3. **Selective Image Processing**: Add filters to skip:
   - Small images (icons, buttons)
   - Ad images
   - Tracking pixels

## Dependencies

### Current Dependencies

- `golang.org/x/net/html`: HTML parsing (Go standard library extension)

### Adding New Dependencies

1. Evaluate necessity - prefer standard library
2. Check maintenance status and popularity
3. Consider size and complexity
4. Add with: `go get <package>`
5. Document in README.md

## Common Issues

### Issue: Ollama Connection Failed

**Solution**: Ensure Ollama is running:
```bash
ollama serve
```

### Issue: Tests Timing Out

**Solution**: Increase timeout in tests or use shorter timeouts in test configs:
```go
config := Config{
    HTTPTimeout: 5 * time.Second,
}
```

### Issue: Image Analysis Fails

**Solution**: Verify vision model is installed:
```bash
ollama pull llama3.2-vision
```

### Issue: Content Extraction Returns Raw Text

**Solution**: This is expected fallback behavior when AI processing fails. Check Ollama logs for errors.

## Release Process

1. Update version in documentation
2. Run full test suite
3. Build for target platforms:
   ```bash
   GOOS=linux GOARCH=amd64 go build -o scraper-linux-amd64
   GOOS=darwin GOARCH=arm64 go build -o scraper-darwin-arm64
   GOOS=windows GOARCH=amd64 go build -o scraper-windows-amd64.exe
   ```
4. Create release notes
5. Tag release: `git tag -a v1.0.0 -m "Release v1.0.0"`

## Contributing

- Follow existing code style
- Write tests for new features
- Update documentation
- Keep commits focused and atomic
- Write clear commit messages

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Ollama API Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html)
- [Testing in Go](https://golang.org/doc/tutorial/add-a-test)
