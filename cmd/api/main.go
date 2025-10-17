package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gurgeh/scraper/api"
	"github.com/gurgeh/scraper/db"
	"github.com/gurgeh/scraper/scraper"
)

func main() {
	// Command-line flags
	addr := flag.String("addr", ":8080", "Server address")
	dbPath := flag.String("db", "scraper.db", "Database file path")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama base URL")
	ollamaModel := flag.String("ollama-model", "llama3.2", "Ollama model to use")
	disableCORS := flag.Bool("disable-cors", false, "Disable CORS")
	flag.Parse()

	// Create server configuration
	config := api.Config{
		Addr: *addr,
		DBConfig: db.Config{
			Driver: "sqlite",
			DSN:    *dbPath,
		},
		ScraperConfig: scraper.Config{
			HTTPTimeout:   30 * time.Second,
			OllamaBaseURL: *ollamaURL,
			OllamaModel:   *ollamaModel,
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
