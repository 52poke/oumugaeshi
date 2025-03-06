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
		// Handle both GET and DELETE requests
		switch r.Method {
		case http.MethodGet:
			handleGetRequest(w, r, s3Client)
		case http.MethodDelete:
			handleDeleteRequest(w, r, s3Client)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleGetRequest processes GET requests for remuxed audio files
func handleGetRequest(w http.ResponseWriter, r *http.Request, s3Client *storage.S3Client) {
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

// handleDeleteRequest processes DELETE requests to remove transcoded files
func handleDeleteRequest(w http.ResponseWriter, r *http.Request, s3Client *storage.S3Client) {
	path := r.URL.Path
	// For DELETE requests, we get the original path and need to find the transcoded path
	// Transform the path to the transcoded format first
	if !strings.Contains(path, "/transcoded/") {
		// Original path format, need to transform it to transcoded path format
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 4 {
			http.Error(w, "Invalid path format", http.StatusBadRequest)
			return
		}

		// Extract filename from path
		filename := pathParts[len(pathParts)-1]

		// Reconstruct the transcoded path
		path = fmt.Sprintf("/wiki/transcoded/%s/%s/%s/%s.webm",
			pathParts[len(pathParts)-3], // hash prefix (e.g. "4")
			pathParts[len(pathParts)-2], // hash second part (e.g. "40")
			filename,                    // filename (e.g. "abc.oga")
			filename)                    // repeated filename with .webm
	}

	// Verify this is for a transcoded file
	if !strings.HasSuffix(path, ".oga.webm") && !strings.HasSuffix(path, ".opus.webm") {
		http.Error(w, "Not a valid transcoded file path", http.StatusBadRequest)
		return
	}

	// Check if the file exists before attempting to delete
	exists, err := s3Client.ObjectExists(path)
	if err != nil {
		log.Printf("Error checking if object exists: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Transcoded file not found", http.StatusNotFound)
		return
	}

	// Delete the transcoded file from S3
	if err := s3Client.DeleteObject(r.Context(), path); err != nil {
		log.Printf("Error deleting transcoded file: %v", err)
		http.Error(w, "Failed to delete transcoded file", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Transcoded file deleted successfully"))
}

// transformPath transforms a path from MediaWiki's transcoded format to the original file path
func transformPath(transcodedPath string) (string, bool) {
	// Extract the file path from the transcoded path pattern
	// Example: /wiki/transcoded/4/40/abc.oga/abc.oga.webm -> /wiki/4/40/abc.oga
	parts := strings.Split(transcodedPath, "/")
	if len(parts) > 6 && parts[1] == "wiki" && parts[2] == "transcoded" && parts[5]+".webm" == parts[6] {
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
