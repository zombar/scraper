package scraper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zombar/scraper/models"
	"github.com/zombar/scraper/ollama"
	"golang.org/x/net/html"
)

// Config contains scraper configuration
type Config struct {
	HTTPTimeout         time.Duration
	OllamaBaseURL       string
	OllamaModel         string
	EnableImageAnalysis bool          // Enable AI-powered image analysis
	MaxImageSizeBytes   int64         // Maximum image size to download (bytes)
	ImageTimeout        time.Duration // Timeout for downloading individual images
}

// DefaultConfig returns default scraper configuration
func DefaultConfig() Config {
	return Config{
		HTTPTimeout:         30 * time.Second,
		OllamaBaseURL:       ollama.DefaultBaseURL,
		OllamaModel:         ollama.DefaultModel,
		EnableImageAnalysis: true,              // Enable image analysis by default
		MaxImageSizeBytes:   10 * 1024 * 1024,  // 10MB max image size
		ImageTimeout:        15 * time.Second,  // 15s timeout per image
	}
}

// Scraper handles web scraping operations
type Scraper struct {
	config       Config
	httpClient   *http.Client
	ollamaClient *ollama.Client
}

// New creates a new Scraper instance
func New(config Config) *Scraper {
	return &Scraper{
		config: config,
		httpClient: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		ollamaClient: ollama.NewClient(config.OllamaBaseURL, config.OllamaModel),
	}
}

// Scrape fetches and processes a URL
func (s *Scraper) Scrape(ctx context.Context, targetURL string) (*models.ScrapedData, error) {
	start := time.Now()

	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL must be http or https")
	}

	// Fetch the page
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Scraper/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	title := extractTitle(doc)
	if title == "" {
		title = targetURL
	}

	// Extract text content
	textContent := extractText(doc)

	// Use Ollama to extract meaningful content
	content, err := s.ollamaClient.ExtractContent(ctx, textContent)
	if err != nil {
		// If Ollama extraction fails, fall back to raw text
		content = textContent
	}

	// Extract images
	images := extractImages(doc, parsedURL)

	// Process images (download and analyze if enabled)
	images = s.processImages(ctx, images)

	// Extract links with Ollama sanitization
	links := s.extractLinksWithOllama(ctx, doc, parsedURL, title, content)

	// Extract metadata
	metadata := extractMetadata(doc)

	// Create scraped data
	data := &models.ScrapedData{
		ID:             uuid.New().String(),
		URL:            targetURL,
		Title:          title,
		Content:        content,
		Images:         images,
		Links:          links,
		FetchedAt:      time.Now(),
		CreatedAt:      time.Now(),
		ProcessingTime: time.Since(start).Seconds(),
		Cached:         false,
		Metadata:       metadata,
	}

	return data, nil
}

// ExtractLinks fetches a URL and returns links using Ollama with fallback to basic extraction
func (s *Scraper) ExtractLinks(ctx context.Context, targetURL string) ([]string, error) {
	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL must be http or https")
	}

	// Fetch the page
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Scraper/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	title := extractTitle(doc)
	if title == "" {
		title = targetURL
	}

	// Extract text content
	textContent := extractText(doc)

	// Use Ollama to extract meaningful content
	content, err := s.ollamaClient.ExtractContent(ctx, textContent)
	if err != nil {
		// If Ollama extraction fails, fall back to raw text
		content = textContent
	}

	// Extract links with Ollama sanitization and fallback
	links := s.extractLinksWithOllama(ctx, doc, parsedURL, title, content)

	return links, nil
}

// extractTitle extracts the page title from the HTML
func extractTitle(n *html.Node) string {
	var title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(title)
}

// extractText extracts all text content from the HTML
func extractText(n *html.Node) string {
	var buf strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString(" ")
			}
		}
		// Skip script and style tags
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(buf.String())
}

// extractImages extracts image information from the HTML
func extractImages(n *html.Node, baseURL *url.URL) []models.ImageInfo {
	var images []models.ImageInfo
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			var src, alt string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "src":
					src = attr.Val
				case "alt":
					alt = attr.Val
				}
			}
			if src != "" {
				// Resolve relative URLs
				if imgURL, err := resolveURL(baseURL, src); err == nil {
					images = append(images, models.ImageInfo{
						URL:     imgURL,
						AltText: alt,
						Summary: "",
						Tags:    []string{},
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return images
}

// extractLinksWithOllama extracts links from HTML and uses Ollama to sanitize them
func (s *Scraper) extractLinksWithOllama(ctx context.Context, n *html.Node, baseURL *url.URL, pageTitle string, pageContent string) []string {
	// First extract all links using the basic method
	allLinks := extractLinks(n, baseURL)

	// Ensure we always return a non-nil slice
	if allLinks == nil {
		allLinks = []string{}
	}

	if len(allLinks) == 0 {
		return allLinks
	}

	// Try to sanitize using Ollama directly
	linksJSON, err := json.Marshal(allLinks)
	if err != nil {
		// If marshaling fails, fall back to returning all links
		return allLinks
	}

	prompt := fmt.Sprintf(`You are a link filtering assistant. Given a list of URLs extracted from a webpage, identify and return ONLY the links that point to substantive content (articles, blog posts, reports, etc.).

INCLUDE:
- Article links (news stories, blog posts, features)
- Opinion pieces and editorials
- Reports, guides, and documentation
- Individual story/content pages
- Links to specific multimedia content (videos, podcasts with their own pages)

EXCLUDE:
- Advertising/sponsored content links
- Site navigation (home, sections, categories, topics)
- Social media share/follow buttons
- Login/signup/account links
- Footer links (privacy, terms, about, contact, jobs, press)
- Newsletter/subscription prompts
- Cookie/consent notices
- Generic section/category/tag pages (unless they're the main content)
- Search functionality links
- Pagination controls (next, previous, page numbers)
- Internal site tools (print, save, bookmark)
- Related external sites/sister publications
- Comment section links

IMPORTANT: If this is a homepage or news aggregator page, it will contain MANY article links - these should ALL be included as they are the primary content. Only filter out the navigation chrome around them.

Page Title: %s

Page Content: %s

Links to filter:
%s

Return ONLY a JSON array of the filtered URLs. Do not include any explanation or commentary.
Format: ["url1", "url2", "url3"]`,
		pageTitle,
		pageContent,
		string(linksJSON))

	response, err := s.ollamaClient.Generate(ctx, prompt)
	if err != nil {
		// If Ollama fails, fall back to returning all links
		return allLinks
	}

	// Parse JSON response
	var sanitizedLinks []string
	if err := json.Unmarshal([]byte(response), &sanitizedLinks); err != nil {
		// If parsing fails, fall back to returning all links
		return allLinks
	}

	// Ensure we never return nil
	if sanitizedLinks == nil {
		sanitizedLinks = []string{}
	}

	return sanitizedLinks
}

// extractLinks extracts links from the HTML
func extractLinks(n *html.Node, baseURL *url.URL) []string {
	var links []string
	seen := make(map[string]bool)
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && attr.Val != "" {
					// Resolve relative URLs
					if linkURL, err := resolveURL(baseURL, attr.Val); err == nil {
						if !seen[linkURL] {
							seen[linkURL] = true
							links = append(links, linkURL)
						}
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return links
}

// extractMetadata extracts page metadata from meta tags
func extractMetadata(n *html.Node) models.PageMetadata {
	metadata := models.PageMetadata{}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, property, content string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "name":
					name = strings.ToLower(attr.Val)
				case "property":
					property = strings.ToLower(attr.Val)
				case "content":
					content = attr.Val
				}
			}

			if content == "" {
				return
			}

			switch {
			case name == "description" || property == "og:description":
				if metadata.Description == "" {
					metadata.Description = content
				}
			case name == "keywords":
				if len(metadata.Keywords) == 0 {
					keywords := strings.Split(content, ",")
					for _, kw := range keywords {
						metadata.Keywords = append(metadata.Keywords, strings.TrimSpace(kw))
					}
				}
			case name == "author" || property == "article:author":
				if metadata.Author == "" {
					metadata.Author = content
				}
			case property == "article:published_time":
				if metadata.PublishedDate == "" {
					metadata.PublishedDate = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return metadata
}

// downloadImage downloads an image from a URL with size and timeout limits
func (s *Scraper) downloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	// Create request with timeout context
	ctx, cancel := context.WithTimeout(ctx, s.config.ImageTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Scraper/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Check content length if available
	if resp.ContentLength > s.config.MaxImageSizeBytes {
		return nil, fmt.Errorf("image too large: %d bytes (max: %d)", resp.ContentLength, s.config.MaxImageSizeBytes)
	}

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, s.config.MaxImageSizeBytes+1)
	imageData, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Check if we exceeded the limit
	if int64(len(imageData)) > s.config.MaxImageSizeBytes {
		return nil, fmt.Errorf("image too large: exceeds %d bytes", s.config.MaxImageSizeBytes)
	}

	return imageData, nil
}

// processImages downloads and analyzes images if image analysis is enabled
func (s *Scraper) processImages(ctx context.Context, images []models.ImageInfo) []models.ImageInfo {
	if !s.config.EnableImageAnalysis {
		log.Printf("Image analysis disabled, returning %d images without analysis", len(images))
		return images
	}

	processedImages := make([]models.ImageInfo, 0, len(images))

	for i, img := range images {
		log.Printf("Processing image %d/%d: %s", i+1, len(images), img.URL)

		// Generate UUID for the image
		img.ID = uuid.New().String()

		// Download the image
		imageData, err := s.downloadImage(ctx, img.URL)
		if err != nil {
			log.Printf("Failed to download image %s: %v", img.URL, err)
			// Keep the image info but without analysis
			processedImages = append(processedImages, img)
			continue
		}

		log.Printf("Downloaded image %s (%d bytes)", img.URL, len(imageData))

		// Store base64 encoded image data
		img.Base64Data = base64.StdEncoding.EncodeToString(imageData)

		// Analyze the image with Ollama
		summary, tags, err := s.ollamaClient.AnalyzeImage(ctx, imageData, img.AltText)
		if err != nil {
			log.Printf("Failed to analyze image %s: %v", img.URL, err)
			// Keep the image info with base64 data but without analysis
			processedImages = append(processedImages, img)
			continue
		}

		// Update image info with analysis results
		img.Summary = summary
		img.Tags = tags
		processedImages = append(processedImages, img)

		log.Printf("Successfully analyzed image %s (summary: %d chars, tags: %d)",
			img.URL, len(summary), len(tags))
	}

	return processedImages
}

// resolveURL resolves a potentially relative URL against a base URL
func resolveURL(base *url.URL, href string) (string, error) {
	// Parse the href
	parsed, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	// Resolve against base
	resolved := base.ResolveReference(parsed)
	return resolved.String(), nil
}
