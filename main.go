package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gurgeh/scraper/scraper"
)

func main() {
	// Command-line flags
	url := flag.String("url", "", "URL to scrape (required)")
	timeout := flag.Duration("timeout", 120*time.Second, "Request timeout")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama base URL")
	ollamaModel := flag.String("ollama-model", "llama3.2", "Ollama model to use")
	pretty := flag.Bool("pretty", false, "Pretty print JSON output")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "Error: -url flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create scraper config
	config := scraper.Config{
		HTTPTimeout:   *timeout,
		OllamaBaseURL: *ollamaURL,
		OllamaModel:   *ollamaModel,
	}

	// Create scraper instance
	s := scraper.New(config)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout*2)
	defer cancel()

	// Scrape the URL
	result, err := s.Scrape(ctx, *url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scraping URL: %v\n", err)
		os.Exit(1)
	}

	// Output JSON
	var jsonData []byte
	if *pretty {
		jsonData, err = json.MarshalIndent(result, "", "  ")
	} else {
		jsonData, err = json.Marshal(result)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}
