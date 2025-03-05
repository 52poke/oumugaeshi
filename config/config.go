package config

import "os"

// Config holds the application configuration
type Config struct {
	S3Bucket    string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Region    string
	ListenAddr  string
}

// LoadFromEnvironment loads configuration from environment variables
func LoadFromEnvironment() Config {
	return Config{
		S3Bucket:    getEnvOrDefault("S3_BUCKET", "mediawiki"),
		S3Endpoint:  getEnvOrDefault("S3_ENDPOINT", "http://localhost:9000"),
		S3AccessKey: getEnvOrDefault("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey: getEnvOrDefault("S3_SECRET_KEY", "minioadmin"),
		S3Region:    getEnvOrDefault("S3_REGION", "us-east-1"),
		ListenAddr:  getEnvOrDefault("LISTEN_ADDR", ":8080"),
	}
}

// getEnvOrDefault retrieves an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
