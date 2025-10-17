# Web Scraper API Server Documentation

The web scraper now includes a full-featured REST API server with persistent storage, batch processing, and CORS support.

## Features

- **Persistent Storage**: SQLite database with automatic migrations
- **Batch Processing**: Process multiple URLs concurrently
- **Caching**: Automatically returns cached results for previously scraped URLs
- **UUID-based IDs**: Each scraped result gets a unique identifier
- **CORS Support**: Ready for web frontend integration
- **Graceful Shutdown**: Handles interrupts cleanly
- **Database Migrations**: Schema versioning for easy upgrades

## Quick Start

### Starting the API Server

```bash
# Default settings (port 8080, scraper.db)
./scraper-api

# Custom port and database
./scraper-api -addr :3000 -db ./data/scraper.db

# Custom Ollama settings
./scraper-api -ollama-url http://localhost:11434 -ollama-model llama3.2

# Disable CORS
./scraper-api -disable-cors
```

### Using Make

```bash
# Build API server
make build-api

# Run API server
make run-api

# Run with custom port
make run-api PORT=3000
```

## API Endpoints

### Health Check

Check if the server is running and get database statistics.

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

Scrape a single URL. Returns cached result if URL was previously scraped.

```http
POST /api/scrape
Content-Type: application/json

{
  "url": "https://example.com",
  "force": false
}
```

**Parameters:**
- `url` (string, required): The URL to scrape
- `force` (boolean, optional): If true, re-scrape even if cached. Default: false

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com",
  "title": "Example Domain",
  "content": "This domain is for use in illustrative examples...",
  "images": [
    {
      "url": "https://example.com/image.jpg",
      "alt_text": "Example image",
      "summary": "A detailed description of the image...",
      "tags": ["example", "illustration", "web"]
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
    "author": "Example Author"
  }
}
```

**Fields:**
- `id`: Unique UUID for this scraped result
- `url`: The URL that was scraped
- `title`: Page title
- `content`: AI-cleaned main content
- `images`: Array of analyzed images
- `links`: All extracted links
- `fetched_at`: When the content was originally fetched
- `created_at`: When this record was first created in the database
- `processing_time_seconds`: Time taken to scrape and process
- `cached`: Boolean indicating if this result was served from cache
- `metadata`: Additional page metadata

**Example:**
```bash
curl -X POST http://localhost:8080/api/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.bbc.co.uk/news/articles/crmxz37nv3zo"}'
```

---

### Batch Scrape

Scrape multiple URLs concurrently (max 50 per request).

```http
POST /api/scrape/batch
Content-Type: application/json

{
  "urls": [
    "https://example.com",
    "https://example.org",
    "https://example.net"
  ],
  "force": false
}
```

**Parameters:**
- `urls` (array of strings, required): URLs to scrape (max 50)
- `force` (boolean, optional): If true, re-scrape even if cached. Default: false

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
    "urls": ["https://example.com", "https://example.org"],
    "force": false
  }'
```

---

### Get by ID

Retrieve scraped data by UUID.

```http
GET /api/data/{id}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://example.com",
  ...
}
```

**Example:**
```bash
curl http://localhost:8080/api/data/550e8400-e29b-41d4-a716-446655440000
```

**Error Response (404):**
```json
{
  "error": "data not found"
}
```

---

### Delete by ID

Delete scraped data by UUID.

```http
DELETE /api/data/{id}
```

**Response:**
```json
{
  "message": "data deleted successfully"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/data/550e8400-e29b-41d4-a716-446655440000
```

**Error Response (404):**
```json
{
  "error": "data not found"
}
```

---

### List All Data

List all scraped data with pagination.

```http
GET /api/data?limit=20&offset=0
```

**Query Parameters:**
- `limit` (integer, optional): Number of results per page (default: 20, max: 100)
- `offset` (integer, optional): Number of results to skip (default: 0)

**Response:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "url": "https://example.com",
      ...
    },
    ...
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

**Example:**
```bash
# Get first page
curl http://localhost:8080/api/data?limit=20&offset=0

# Get second page
curl http://localhost:8080/api/data?limit=20&offset=20
```

---

## Database

### Schema

The database uses SQLite by default but is designed for easy PostgreSQL migration.

**scraped_data table:**
```sql
CREATE TABLE scraped_data (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Indexes:**
- `idx_scraped_data_url` on `url`
- `idx_scraped_data_created_at` on `created_at`

### Migrations

Migrations are automatically applied on startup. The system uses a `schema_migrations` table to track applied migrations.

**View migration status:**
```bash
# Connect to database
sqlite3 scraper.db "SELECT * FROM schema_migrations"
```

**Migrations are versioned:**
- Version 1: Create scraped_data table
- Version 2: Create schema_migrations table

To add new migrations, edit `db/migrations.go` and add a new migration to the `migrations` slice.

### Switching to PostgreSQL

To use PostgreSQL instead of SQLite:

1. Install PostgreSQL driver:
   ```bash
   go get github.com/lib/pq
   ```

2. Update imports in `db/db.go`:
   ```go
   import _ "github.com/lib/pq"
   ```

3. Start server with PostgreSQL DSN:
   ```bash
   ./scraper-api -db "postgres://user:pass@localhost/scraper?sslmode=disable"
   ```

4. Update SQL statements in `db/migrations.go` for PostgreSQL syntax if needed.

---

## Configuration

### Command-Line Flags

```bash
./scraper-api [flags]
```

**Flags:**
- `-addr string`: Server address (default ":8080")
- `-db string`: Database file path (default "scraper.db")
- `-ollama-url string`: Ollama base URL (default "http://localhost:11434")
- `-ollama-model string`: Ollama model to use (default "llama3.2")
- `-disable-cors`: Disable CORS (enabled by default)

### Environment-Based Config (Future)

For production deployments, consider adding environment variable support:

```bash
export SCRAPER_ADDR=":8080"
export SCRAPER_DB="scraper.db"
export SCRAPER_OLLAMA_URL="http://localhost:11434"
```

---

## Error Handling

All errors return JSON with an `error` field:

```json
{
  "error": "descriptive error message"
}
```

**Common HTTP Status Codes:**
- `200 OK`: Success
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `405 Method Not Allowed`: Wrong HTTP method
- `500 Internal Server Error`: Server error

---

## Performance Considerations

### Batch Processing

- Maximum 50 URLs per batch request
- URLs are processed concurrently using goroutines
- Each URL has a 2-minute timeout
- Failed URLs don't affect successful ones

### Caching

- URLs are deduplicated using database UNIQUE constraint
- Subsequent requests for the same URL return cached results instantly
- Use `"force": true` to bypass cache
- The `cached` field indicates if the result was served from cache
- The `created_at` field shows when the data was originally scraped
- Use `fetched_at` and `created_at` together to detect stale data

**Cache Staleness Example:**
```javascript
// Check if data is older than 24 hours
const response = await fetch('http://localhost:8080/api/scrape', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ url: 'https://example.com' })
});

const data = await response.json();
const ageInHours = (Date.now() - new Date(data.created_at)) / (1000 * 60 * 60);

if (data.cached && ageInHours > 24) {
  console.log('Data is stale, re-scraping...');
  // Re-scrape with force
  const freshData = await fetch('http://localhost:8080/api/scrape', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url: 'https://example.com', force: true })
  });
}
```

### Database

- Connection pool: 25 max open connections, 5 idle
- Connection lifetime: 5 minutes
- Uses prepared statements for SQL queries
- Indexes on url and created_at for fast lookups

### Timeouts

- HTTP read timeout: 30 seconds
- HTTP write timeout: 120 seconds (for long scraping operations)
- Idle timeout: 120 seconds
- Scraping timeout: 2 minutes per URL

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

// List all
async function listAll(limit = 20, offset = 0): Promise<ListResponse> {
  const response = await fetch(
    `http://localhost:8080/api/data?limit=${limit}&offset=${offset}`
  );
  return response.json();
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
```

### cURL Examples

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

## Production Deployment

### Systemd Service

Create `/etc/systemd/system/scraper-api.service`:

```ini
[Unit]
Description=Web Scraper API
After=network.target

[Service]
Type=simple
User=scraper
WorkingDirectory=/opt/scraper
ExecStart=/opt/scraper/scraper-api -addr :8080 -db /var/lib/scraper/scraper.db
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable scraper-api
sudo systemctl start scraper-api
sudo systemctl status scraper-api
```

### Docker

Create `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o scraper-api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/scraper-api .
EXPOSE 8080
CMD ["./scraper-api"]
```

Build and run:

```bash
docker build -t scraper-api .
docker run -p 8080:8080 -v $(pwd)/data:/data scraper-api -db /data/scraper.db
```

### Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name scraper.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 180s;
    }
}
```

---

## Troubleshooting

### Database locked errors

If you see "database is locked" errors with SQLite:
- Ensure only one instance is running
- Check file permissions
- Consider using PostgreSQL for multi-instance deployments

### Ollama connection errors

```
failed to scrape: failed to extract content: failed to send request
```

**Solution:** Ensure Ollama is running:
```bash
ollama serve
```

### Port already in use

```
listen tcp :8080: bind: address already in use
```

**Solution:** Use a different port:
```bash
./scraper-api -addr :3000
```

### Out of memory

For large batch operations, increase system resources or reduce batch size.

---

## Security Considerations

- **Input Validation**: URLs are validated before scraping
- **Rate Limiting**: Not implemented - add middleware if needed
- **Authentication**: Not implemented - add if exposing publicly
- **CORS**: Enabled by default, disable with `-disable-cors` for internal use
- **SQL Injection**: Protected by using prepared statements
- **XSS**: Content is stored as-is, sanitize in frontend

For production use, consider adding:
- API key authentication
- Rate limiting middleware
- Request size limits
- IP whitelisting
- HTTPS/TLS support
