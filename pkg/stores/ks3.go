package stores

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	ks3aws "github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/credentials"
	ks3s3 "github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// Ks3Store implements Store for Kingsoft Cloud KS3.
// KS3 uses V2 signing and virtual-hosted style addressing.
type Ks3Store struct {
	Endpoint        string `env:"KS3_ENDPOINT"`
	Region          string `env:"KS3_REGION"`
	AccessKeyID     string `env:"KS3_ACCESS_KEY_ID"`
	AccessKeySecret string `env:"KS3_SECRET_ACCESS_KEY"`
	BucketName      string `env:"KS3_BUCKET"`
	Domain          string `env:"KS3_DOMAIN"` // Optional custom domain for public access
}

// NewKs3Store creates a KS3 storage instance from environment variables.
func NewKs3Store() Store {
	return &Ks3Store{
		Endpoint:        utils.GetEnv("KS3_ENDPOINT"),
		Region:          utils.GetEnv("KS3_REGION"),
		AccessKeyID:     utils.GetEnv("KS3_ACCESS_KEY_ID"),
		AccessKeySecret: utils.GetEnv("KS3_SECRET_ACCESS_KEY"),
		BucketName:      utils.GetEnv("KS3_BUCKET"),
		Domain:          utils.GetEnv("KS3_DOMAIN"),
	}
}

func (s *Ks3Store) client() *ks3s3.S3 {
	creds := credentials.NewStaticCredentials(s.AccessKeyID, s.AccessKeySecret, "")
	return ks3s3.New(&ks3aws.Config{
		Credentials:      creds,
		Region:           s.Region,
		Endpoint:         s.Endpoint,
		DisableSSL:       false,
		S3ForcePathStyle: false,
		SignerVersion:    "V2",
		MaxRetries:       3,
	})
}

// Read reads a file from KS3.
func (s *Ks3Store) Read(key string) (io.ReadCloser, int64, error) {
	client := s.client()
	result, err := client.GetObject(&ks3s3.GetObjectInput{
		Bucket: ks3aws.String(s.BucketName),
		Key:    ks3aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object from KS3: %w", err)
	}

	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return result.Body, size, nil
}

// Write writes a file to KS3.
func (s *Ks3Store) Write(key string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read upload body: %w", err)
	}

	client := s.client()
	_, err = client.PutObject(&ks3s3.PutObjectInput{
		Bucket: ks3aws.String(s.BucketName),
		Key:    ks3aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object to KS3: %w", err)
	}
	return nil
}

// Delete deletes a file from KS3.
func (s *Ks3Store) Delete(key string) error {
	client := s.client()
	_, err := client.DeleteObject(&ks3s3.DeleteObjectInput{
		Bucket: ks3aws.String(s.BucketName),
		Key:    ks3aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from KS3: %w", err)
	}
	return nil
}

// Exists checks if a file exists in KS3.
func (s *Ks3Store) Exists(key string) (bool, error) {
	client := s.client()
	_, err := client.HeadObject(&ks3s3.HeadObjectInput{
		Bucket: ks3aws.String(s.BucketName),
		Key:    ks3aws.String(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence in KS3: %w", err)
	}
	return true, nil
}

// PublicURL returns the public URL for a file in KS3.
func (s *Ks3Store) PublicURL(key string) string {
	key = strings.TrimPrefix(key, "/")
	if s.Domain != "" {
		domain := strings.TrimSuffix(s.Domain, "/")
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		return fmt.Sprintf("%s/%s", domain, key)
	}
	endpoint := strings.TrimPrefix(strings.TrimPrefix(s.Endpoint, "https://"), "http://")
	return fmt.Sprintf("https://%s.%s/%s", s.BucketName, endpoint, key)
}

// CheckKs3Connectivity tests KS3 connectivity using the provided credentials.
func CheckKs3Connectivity(ctx context.Context, endpoint, region, accessKey, secretKey, bucketName string) error {
	creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
	client := ks3s3.New(&ks3aws.Config{
		Credentials:      creds,
		Region:           region,
		Endpoint:         endpoint,
		DisableSSL:       false,
		S3ForcePathStyle: false,
		SignerVersion:    "V2",
	})

	done := make(chan error, 1)
	go func() {
		_, err := client.HeadBucket(&ks3s3.HeadBucketInput{
			Bucket: ks3aws.String(bucketName),
		})
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("KS3 connectivity check failed: %w", err)
		}
		return nil
	}
}
