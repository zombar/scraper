# Scraper API Reference

REST API documentation for the web scraper service.

## Base URL

```
http://localhost:8080
```

## Endpoints

### Health Check

Check server status and get database statistics.

**Request:**
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "count": 42,
  "time": "2024-01-15T14:23:45Z"
}
```

---

### Scrape Single URL

Scrape a single URL. Returns cached result if previously scraped.

**Request:**
```http
POST /api/scrape
Content-Type: application/json

{
  "url": "https://example.com",
  "force": false
}
```

**Parameters:**
- `url` (string, required) - URL to scrape
- `force` (boolean, optional) - Bypass cache and re-scrape (default: false)

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com",
  "title": "Example Domain",
  "content": "AI-cleaned main content...",
  "images": [
    {
      "url": "https://example.com/image.jpg",
      "alt_text": "Example image",
      "summary": "AI-generated description...",
      "tags": ["example", "illustration"]
    }
  ],
  "links": [
    "https://example.com/about",
    "https://example.com/contact"
  ],
  "fetched_at": "2024-01-15T14:23:45Z",
  "created_at": "2024-01-15T14:23:45Z",
  "processing_time_seconds": 8.34,
  "cached": false,
  "metadata": {
    "description": "Example domain description",
    "keywords": ["example", "domain"],
    "author": "Example Author",
    "published_date": "2024-01-15"
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

---

### Batch Scrape

Scrape multiple URLs concurrently (maximum 50 per request).

**Request:**
```http
POST /api/scrape/batch
Content-Type: application/json

{
  "urls": [
    "https://example.com",
    "https://example.org"
  ],
  "force": false
}
```

**Parameters:**
- `urls` (array of strings, required) - URLs to scrape (max 50)
- `force` (boolean, optional) - Bypass cache for all URLs (default: false)

**Response:**
```json
{
  "results": [
    {
      "url": "https://example.com",
      "success": true,
      "data": { ... },
      "cached": true
    },
    {
      "url": "https://example.org",
      "success": true,
      "data": { ... },
      "cached": false
    },
    {
      "url": "https://invalid-url",
      "success": false,
      "error": "failed to fetch page",
      "cached": false
    }
  ],
  "summary": {
    "total": 3,
    "success": 2,
    "failed": 1,
    "cached": 1,
    "scraped": 1
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/scrape/batch \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://example.com", "https://example.org"]
  }'
```

---

### Get by ID

Retrieve scraped data by UUID.

**Request:**
```http
GET /api/data/{id}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com",
  "title": "Page Title",
  ...
}
```

**Error Response (404):**
```json
{
  "error": "data not found"
}
```

**Example:**
```bash
curl http://localhost:8080/api/data/550e8400-e29b-41d4-a716-446655440000
```

---

### Delete by ID

Delete scraped data by UUID.

**Request:**
```http
DELETE /api/data/{id}
```

**Response:**
```json
{
  "message": "data deleted successfully"
}
```

**Error Response (404):**
```json
{
  "error": "data not found"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/data/550e8400-e29b-41d4-a716-446655440000
```

---

### Get Image by ID

Retrieve a specific image by its UUID.

**Request:**
```http
GET /api/images/{id}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com/image.jpg",
  "alt_text": "Example image",
  "summary": "AI-generated 4-5 sentence description of the image...",
  "tags": ["example", "illustration", "diagram"],
  "base64_data": "iVBORw0KGgoAAAANSUhEUgAAAAEA..."
}
```

**Error Response (404):**
```json
{
  "error": "image not found"
}
```

**Example:**
```bash
curl http://localhost:8080/api/images/550e8400-e29b-41d4-a716-446655440000
```

---

### Search Images by Tags

Search for images using fuzzy tag matching (case-insensitive substring matching).

**Request:**
```http
POST /api/images/search
Content-Type: application/json

{
  "tags": ["cat", "animal"]
}
```

**Parameters:**
- `tags` (array of strings, required) - Tags to search for (fuzzy matching)

**Response:**
```json
{
  "images": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "url": "https://example.com/cat.jpg",
      "alt_text": "A cat photo",
      "summary": "Image shows a domestic cat...",
      "tags": ["cat", "animal", "pet"],
      "base64_data": "iVBORw0KGgoAAAANSUhEUgAAAAEA..."
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "url": "https://example.com/wildlife.jpg",
      "alt_text": "Wildlife scene",
      "summary": "Image depicts various animals in nature...",
      "tags": ["animals", "wildlife", "nature"],
      "base64_data": "iVBORw0KGgoAAAANSUhEUgAAAAEA..."
    }
  ],
  "count": 2
}
```

**Fuzzy Matching:** Searches are case-insensitive and match substrings. For example:
- Searching for "cat" will match images with tags: "cat", "cats", "wildcat", "scatter"
- Searching for "anim" will match images with tags: "animal", "animation", "animals"

**Example:**
```bash
curl -X POST http://localhost:8080/api/images/search \
  -H "Content-Type: application/json" \
  -d '{"tags": ["cat", "dog"]}'
```

---

### List All Data

List all scraped data with pagination.

**Request:**
```http
GET /api/data?limit=20&offset=0
```

**Query Parameters:**
- `limit` (integer, optional) - Results per page (default: 20, max: 100)
- `offset` (integer, optional) - Number of results to skip (default: 0)

**Response:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "url": "https://example.com",
      ...
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

**Example:**
```bash
# First page
curl "http://localhost:8080/api/data?limit=20&offset=0"

# Second page
curl "http://localhost:8080/api/data?limit=20&offset=20"
```

---

## Data Types

### ScrapedData

Main output structure containing all scraped content.

```go
type ScrapedData struct {
    ID              string        `json:"id"`
    URL             string        `json:"url"`
    Title           string        `json:"title"`
    Content         string        `json:"content"`
    Images          []ImageInfo   `json:"images"`
    Links           []string      `json:"links"`
    FetchedAt       time.Time     `json:"fetched_at"`
    CreatedAt       time.Time     `json:"created_at"`
    ProcessingTime  float64       `json:"processing_time_seconds"`
    Cached          bool          `json:"cached"`
    Metadata        PageMetadata  `json:"metadata"`
}
```

**Fields:**
- `id` - Unique UUID identifier
- `url` - Scraped URL
- `title` - Page title from `<title>` tag
- `content` - AI-cleaned main content
- `images` - Array of image information
- `links` - All extracted hyperlinks
- `fetched_at` - When content was originally fetched
- `created_at` - When record was created in database
- `processing_time_seconds` - Total processing time
- `cached` - Whether result was served from cache
- `metadata` - Additional page metadata

### ImageInfo

Information about an extracted image.

```go
type ImageInfo struct {
    ID         string   `json:"id,omitempty"`
    URL        string   `json:"url"`
    AltText    string   `json:"alt_text"`
    Summary    string   `json:"summary"`
    Tags       []string `json:"tags"`
    Base64Data string   `json:"base64_data,omitempty"`
}
```

**Fields:**
- `id` - Unique UUID identifier for the image
- `url` - Absolute image URL
- `alt_text` - Alt text from `<img>` tag
- `summary` - AI-generated 4-5 sentence description
- `tags` - AI-generated tags for categorization
- `base64_data` - Base64-encoded image data (omitted in list responses for performance)

### PageMetadata

Metadata extracted from HTML meta tags.

```go
type PageMetadata struct {
    Description   string   `json:"description,omitempty"`
    Keywords      []string `json:"keywords,omitempty"`
    Author        string   `json:"author,omitempty"`
    PublishedDate string   `json:"published_date,omitempty"`
}
```

---

## Error Responses

All errors return JSON with an `error` field:

```json
{
  "error": "descriptive error message"
}
```

**HTTP Status Codes:**
- `200 OK` - Success
- `400 Bad Request` - Invalid request parameters
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

---

## Integration Examples

### JavaScript/TypeScript

```typescript
// Scrape single URL
async function scrapeURL(url: string): Promise<ScrapedData> {
  const response = await fetch('http://localhost:8080/api/scrape', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url })
  });
  return response.json();
}

// Batch scrape
async function scrapeBatch(urls: string[]): Promise<BatchResponse> {
  const response = await fetch('http://localhost:8080/api/scrape/batch', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ urls })
  });
  return response.json();
}

// Get by ID
async function getByID(id: string): Promise<ScrapedData> {
  const response = await fetch(`http://localhost:8080/api/data/${id}`);
  return response.json();
}

// List all with pagination
async function listAll(limit = 20, offset = 0): Promise<ListResponse> {
  const response = await fetch(
    `http://localhost:8080/api/data?limit=${limit}&offset=${offset}`
  );
  return response.json();
}

// Check cache freshness
async function checkCacheFreshness(url: string): Promise<void> {
  const data = await scrapeURL(url);
  const ageInHours = (Date.now() - new Date(data.created_at).getTime()) / (1000 * 60 * 60);

  if (data.cached && ageInHours > 24) {
    console.log('Data is stale, re-scraping...');
    const freshData = await fetch('http://localhost:8080/api/scrape', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url, force: true })
    }).then(res => res.json());
  }
}
```

### Python

```python
import requests

# Scrape single URL
def scrape_url(url: str, force: bool = False) -> dict:
    response = requests.post(
        'http://localhost:8080/api/scrape',
        json={'url': url, 'force': force}
    )
    return response.json()

# Batch scrape
def scrape_batch(urls: list[str], force: bool = False) -> dict:
    response = requests.post(
        'http://localhost:8080/api/scrape/batch',
        json={'urls': urls, 'force': force}
    )
    return response.json()

# Get by ID
def get_by_id(id: str) -> dict:
    response = requests.get(f'http://localhost:8080/api/data/{id}')
    return response.json()

# Delete by ID
def delete_by_id(id: str) -> dict:
    response = requests.delete(f'http://localhost:8080/api/data/{id}')
    return response.json()

# List with pagination
def list_all(limit: int = 20, offset: int = 0) -> dict:
    response = requests.get(
        f'http://localhost:8080/api/data?limit={limit}&offset={offset}'
    )
    return response.json()
```

### cURL

```bash
# Health check
curl http://localhost:8080/health

# Scrape URL
curl -X POST http://localhost:8080/api/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# Force re-scrape
curl -X POST http://localhost:8080/api/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "force": true}'

# Batch scrape
curl -X POST http://localhost:8080/api/scrape/batch \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com", "https://example.org"]}'

# Get by ID
curl http://localhost:8080/api/data/550e8400-e29b-41d4-a716-446655440000

# List with pagination
curl "http://localhost:8080/api/data?limit=10&offset=0"

# Delete by ID
curl -X DELETE http://localhost:8080/api/data/550e8400-e29b-41d4-a716-446655440000
```

---

## Configuration

### Command-Line Flags

```bash
./scraper-api [flags]
```

- `-addr string` - Server address (default: ":8080")
- `-db string` - Database file path (default: "scraper.db")
- `-ollama-url string` - Ollama base URL (default: "http://localhost:11434")
- `-ollama-model string` - Ollama model (default: "llama3.2")
- `-disable-cors` - Disable CORS (enabled by default)

### Environment Variables

For production deployments, use environment variables:

```bash
export SCRAPER_ADDR=":8080"
export SCRAPER_DB="scraper.db"
export SCRAPER_OLLAMA_URL="http://localhost:11434"
export SCRAPER_OLLAMA_MODEL="llama3.2"
```

---

## Performance

### Batch Processing

- Maximum 50 URLs per batch request
- URLs processed concurrently using goroutines
- Each URL has 2-minute timeout
- Failed URLs don't affect successful ones

### Caching

- URLs deduplicated using database unique constraint
- Cached results returned instantly
- Use `force: true` to bypass cache
- `cached` field indicates cache status
- `created_at` shows original scrape time

### Database

- Connection pool: 25 max open, 5 idle
- Connection lifetime: 5 minutes
- Prepared statements for queries
- Indexes on url and created_at

### Timeouts

- HTTP read timeout: 30 seconds
- HTTP write timeout: 120 seconds
- Idle timeout: 120 seconds
- Scraping timeout: 2 minutes per URL

---

## Database Schema

### scraped_data Table

```sql
CREATE TABLE scraped_data (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### images Table

Images are stored separately from scraped data for efficient querying and retrieval.

```sql
CREATE TABLE images (
    id TEXT PRIMARY KEY,
    scrape_id TEXT NOT NULL,
    url TEXT NOT NULL,
    alt_text TEXT,
    summary TEXT,
    tags TEXT,
    base64_data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (scrape_id) REFERENCES scraped_data(id) ON DELETE CASCADE
);
```

**Note:** The `tags` field stores a JSON array of strings. Images are automatically deleted when their parent scraped data is deleted (cascade delete).

### Indexes

**scraped_data:**
- `idx_scraped_data_url` on `url`
- `idx_scraped_data_created_at` on `created_at`

**images:**
- `idx_images_scrape_id` on `scrape_id`
- `idx_images_created_at` on `created_at`

### Migrations

Migrations are automatically applied on startup using a version-based system tracked in the `schema_migrations` table.

To add new migrations, edit `db/migrations.go` and add to the `migrations` slice.
