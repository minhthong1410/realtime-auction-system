package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kurama/auction-system/backend/internal/config"
)

type S3Client struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewS3Client(cfg config.StorageConfig) *S3Client {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.Endpoint),
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
	})

	return &S3Client{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
	}
}

// Upload uploads a file and returns the public URL.
func (s *S3Client) Upload(ctx context.Context, folder string, filename string, contentType string, body io.Reader) (string, error) {
	key := path.Join(folder, fmt.Sprintf("%d_%s", time.Now().UnixMilli(), filename))

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.publicURL, key), nil
}
