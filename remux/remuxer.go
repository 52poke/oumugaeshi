package remux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/52poke/oumugaeshi/storage"
	"github.com/google/uuid"
)

// RemuxAndStore downloads a file from S3, remuxes it to WebM, and uploads it back
func RemuxAndStore(s3Client *storage.S3Client, sourcePath, destPath string) error {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("oumugaeshi-%s", uuid.New().String()))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download the source file
	sourceFile := filepath.Join(tempDir, "source")
	if err := s3Client.DownloadFile(sourcePath, sourceFile); err != nil {
		return fmt.Errorf("failed to download source file: %w", err)
	}

	// Create output file path
	outputFile := filepath.Join(tempDir, "output.webm")

	// Remux the file to WebM container
	if err := remuxToWebM(sourceFile, outputFile); err != nil {
		return fmt.Errorf("remuxing failed: %w", err)
	}

	// Upload the remuxed file to S3
	if err := s3Client.UploadFile(outputFile, destPath, "audio/webm"); err != nil {
		return fmt.Errorf("failed to upload remuxed file: %w", err)
	}

	return nil
}

// remuxToWebM remuxes a media file to WebM container format without transcoding
func remuxToWebM(sourceFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-i", sourceFile,
		"-c", "copy", // Copy without transcoding
		"-f", "webm", // Force WebM container format
		outputFile)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return nil
}
