package main

import (
	"log"
	"net/http"

	"github.com/52poke/oumugaeshi/config"
	"github.com/52poke/oumugaeshi/handler"
	"github.com/52poke/oumugaeshi/storage"
)

func main() {
	// Load configuration from environment
	cfg := config.LoadFromEnvironment()

	// Initialize S3 client
	s3Client, err := storage.NewS3Client(cfg.S3Endpoint, cfg.S3Region, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// Create HTTP handler
	proxyHandler := handler.NewProxyHandler(s3Client)

	// Start the HTTP server
	log.Printf("Starting server on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, proxyHandler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
