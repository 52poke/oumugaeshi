package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/52poke/oumugaeshi/remux"
	"github.com/52poke/oumugaeshi/storage"
)

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(s3Client *storage.S3Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if the request is for a .webm file that needs remuxing
		path := r.URL.Path
		if !strings.HasSuffix(path, ".oga.webm") && !strings.HasSuffix(path, ".opus.webm") {
			http.Error(w, "Not a .webm remux request", http.StatusBadRequest)
			return
		}

		// Check if the remuxed file exists in S3
		exists, err := s3Client.ObjectExists(path)
		if err != nil {
			log.Printf("Error checking if object exists: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if exists {
			// If it exists, serve it from S3
			serveFromS3(w, r, s3Client, path)
			return
		}

		// Check if original file exists in different location based on pattern
		// From: /wiki/transcoded/4/40/abc.oga/abc.oga.webm
		// To:   /wiki/4/40/abc.oga
		originalPath, ok := transformPath(path)
		if !ok {
			http.Error(w, "Invalid path format", http.StatusBadRequest)
			return
		}
		log.Println("Original path:", originalPath)

		// Check if original file exists
		exists, err = s3Client.ObjectExists(originalPath)
		if err != nil {
			log.Printf("Error checking if original object exists: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, "Original file not found", http.StatusNotFound)
			return
		}

		// Remux the file and store it in S3
		if err := remux.RemuxAndStore(s3Client, originalPath, path); err != nil {
			log.Printf("Error remuxing file: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Serve the newly remuxed file
		serveFromS3(w, r, s3Client, path)
	}
}

// transformPath transforms a path from MediaWiki's transcoded format to the original file path
func transformPath(transcodedPath string) (string, bool) {
	// Extract the file path from the transcoded path pattern
	// Example: /wiki/transcoded/4/40/abc.oga/abc.oga.webm -> /wiki/4/40/abc.oga
	parts := strings.Split(transcodedPath, "/")
	if len(parts) >= 6 && parts[1] == "wiki" && parts[2] == "transcoded" && parts[5]+".webm" == parts[6] {
		// Reconstruct path without the "transcoded" part and the last segment
		return fmt.Sprintf("/%s/%s/%s/%s", parts[1], parts[3], parts[4], parts[5]), true
	}
	return "", false
}

// serveFromS3 serves a file directly from S3
func serveFromS3(w http.ResponseWriter, r *http.Request, s3Client *storage.S3Client, path string) {
	ctx := r.Context()

	// Get object from S3
	output, err := s3Client.GetObject(ctx, path)
	if err != nil {
		log.Printf("Error getting object from S3: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer output.Body.Close()

	// Set content type header
	w.Header().Set("Content-Type", "audio/webm")
	if output.ContentLength != nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", *output.ContentLength))
	}
	w.Header().Set("Cache-Control", "max-age=86400") // Cache for 24 hours

	// Copy the file to the response
	if _, err := io.Copy(w, output.Body); err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}
