package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/zombar/scraper"
	"github.com/zombar/scraper/api"
	"github.com/zombar/scraper/db"
)

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Default values
	defaultPort := getEnv("PORT", "8080")
	defaultDBPath := getEnv("DB_PATH", "scraper.db")
	defaultOllamaURL := getEnv("OLLAMA_URL", "http://localhost:11434")
	defaultOllamaModel := getEnv("OLLAMA_MODEL", "gpt-oss:20b")
	defaultLinkScoreThreshold := getEnv("LINK_SCORE_THRESHOLD", "0.5")

	// Parse link score threshold
	linkScoreThreshold, err := strconv.ParseFloat(defaultLinkScoreThreshold, 64)
	if err != nil {
		log.Printf("Invalid LINK_SCORE_THRESHOLD value, using default 0.5: %v", err)
		linkScoreThreshold = 0.5
	}

	// Command-line flags (override environment variables)
	port := flag.String("port", defaultPort, "Server port")
	dbPath := flag.String("db", defaultDBPath, "Database file path")
	ollamaURL := flag.String("ollama-url", defaultOllamaURL, "Ollama base URL")
	ollamaModel := flag.String("ollama-model", defaultOllamaModel, "Ollama model to use")
	scoreThreshold := flag.Float64("link-score-threshold", linkScoreThreshold, "Minimum score for link recommendation (0.0-1.0)")
	disableCORS := flag.Bool("disable-cors", false, "Disable CORS")
	disableImageAnalysis := flag.Bool("disable-image-analysis", false, "Disable AI-powered image analysis")
	flag.Parse()

	// Create server configuration
	config := api.Config{
		Addr: ":" + *port,
		DBConfig: db.Config{
			Driver: "sqlite",
			DSN:    *dbPath,
		},
		ScraperConfig: scraper.Config{
			HTTPTimeout:         30 * time.Second,
			OllamaBaseURL:       *ollamaURL,
			OllamaModel:         *ollamaModel,
			EnableImageAnalysis: !*disableImageAnalysis,
			MaxImageSizeBytes:   10 * 1024 * 1024, // 10MB
			ImageTimeout:        15 * time.Second,
			LinkScoreThreshold:  *scoreThreshold,
		},
		CORSEnabled: !*disableCORS,
	}

	// Create server
	server, err := api.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	fmt.Println("\nShutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	fmt.Println("Server stopped")
}
