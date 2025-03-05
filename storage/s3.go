package storage

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps the S3 client functionality
type S3Client struct {
	Client     *s3.Client
	BucketName string
	Endpoint   string
}

// NewS3Client creates a new S3 client
func NewS3Client(endpoint, region, bucketName, accessKey, secretKey string) (*S3Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	return &S3Client{
		Client:     s3.NewFromConfig(cfg),
		BucketName: bucketName,
		Endpoint:   endpoint,
	}, nil
}

// ObjectExists checks if an object exists in the bucket
func (s *S3Client) ObjectExists(path string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key := strings.TrimPrefix(path, "/")
	_, err := s.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if error is because object doesn't exist
		if strings.Contains(err.Error(), "NotFound") ||
			strings.Contains(err.Error(), "NoSuchKey") ||
			strings.Contains(err.Error(), "404") ||
			strings.Contains(err.Error(), "Forbidden") ||
			strings.Contains(err.Error(), "403") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetObject downloads an object from S3
func (s *S3Client) GetObject(ctx context.Context, path string) (*s3.GetObjectOutput, error) {
	return s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(strings.TrimPrefix(path, "/")),
	})
}

// DownloadFile downloads a file from S3 to a local path
func (s *S3Client) DownloadFile(s3Path, localPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	output, err := s.GetObject(ctx, s3Path)
	if err != nil {
		return err
	}
	defer output.Body.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, output.Body)
	return err
}

// UploadFile uploads a local file to S3
func (s *S3Client) UploadFile(localPath, s3Path string, contentType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.BucketName),
		Key:         aws.String(strings.TrimPrefix(s3Path, "/")),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	return err
}
