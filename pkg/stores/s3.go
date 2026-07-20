package stores

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Store represents Amazon S3 storage
type S3Store struct {
	Region          string `env:"S3_REGION"`
	AccessKeyID     string `env:"S3_ACCESS_KEY_ID"`
	AccessKeySecret string `env:"S3_SECRET_ACCESS_KEY"`
	BucketName      string `env:"S3_BUCKET"`
	Endpoint        string `env:"S3_ENDPOINT"` // Custom endpoint for S3-compatible services
	UsePathStyle    bool   `env:"S3_USE_PATH_STYLE"`
	Domain          string `env:"S3_DOMAIN"` // Custom domain for public access
}

// NewS3Store creates a new S3 storage instance
func NewS3Store() Store {
	usePathStyle := strings.ToLower(utils.GetEnv("S3_USE_PATH_STYLE")) == "true" || utils.GetEnv("S3_USE_PATH_STYLE") == "1"
	return &S3Store{
		Region:          utils.GetEnv("S3_REGION"),
		AccessKeyID:     utils.GetEnv("S3_ACCESS_KEY_ID"),
		AccessKeySecret: utils.GetEnv("S3_SECRET_ACCESS_KEY"),
		BucketName:      utils.GetEnv("S3_BUCKET"),
		Endpoint:        utils.GetEnv("S3_ENDPOINT"),
		UsePathStyle:    usePathStyle,
		Domain:          utils.GetEnv("S3_DOMAIN"),
	}
}

// client returns an S3 client
func (s *S3Store) client(ctx context.Context) (*s3.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s.AccessKeyID, s.AccessKeySecret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If custom endpoint is provided, use it (for S3-compatible services)
	options := []func(*s3.Options){}
	if s.Endpoint != "" {
		options = append(options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s.Endpoint)
			o.UsePathStyle = s.UsePathStyle
		})
	}

	return s3.NewFromConfig(cfg, options...), nil
}

// Read reads a file from S3
func (s *S3Store) Read(key string) (io.ReadCloser, int64, error) {
	ctx := context.Background()
	client, err := s.client(ctx)
	if err != nil {
		return nil, 0, err
	}

	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object: %w", err)
	}

	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return result.Body, size, nil
}

// Write writes a file to S3
func (s *S3Store) Write(key string, r io.Reader) error {
	ctx := context.Background()
	client, err := s.client(ctx)
	if err != nil {
		return err
	}

	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

// Delete deletes a file from S3
func (s *S3Store) Delete(key string) error {
	ctx := context.Background()
	client, err := s.client(ctx)
	if err != nil {
		return err
	}

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// Exists checks if a file exists in S3
func (s *S3Store) Exists(key string) (bool, error) {
	ctx := context.Background()
	client, err := s.client(ctx)
	if err != nil {
		return false, err
	}

	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// PublicURL returns the public URL for a file
func (s *S3Store) PublicURL(key string) string {
	// If custom domain is set, use it
	if s.Domain != "" {
		domain := strings.TrimSuffix(s.Domain, "/")
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		return fmt.Sprintf("%s/%s", domain, strings.TrimPrefix(key, "/"))
	}
	if s.Endpoint != "" {
		endpoint := strings.TrimSuffix(s.Endpoint, "/")
		if s.UsePathStyle {
			return fmt.Sprintf("%s/%s/%s", endpoint, s.BucketName, strings.TrimPrefix(key, "/"))
		}
		return fmt.Sprintf("%s/%s", endpoint, strings.TrimPrefix(key, "/"))
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.BucketName, s.Region, strings.TrimPrefix(key, "/"))
}
