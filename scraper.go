package scraper

import (
	"context"
	"fmt"
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
	HTTPTimeout   time.Duration
	OllamaBaseURL string
	OllamaModel   string
}

// DefaultConfig returns default scraper configuration
func DefaultConfig() Config {
	return Config{
		HTTPTimeout:   30 * time.Second,
		OllamaBaseURL: ollama.DefaultBaseURL,
		OllamaModel:   ollama.DefaultModel,
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

	// Extract links
	links := extractLinks(doc, parsedURL)

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
