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

// ObsStore implements Store for Huawei Cloud OBS (S3-compatible).
type ObsStore struct {
	Endpoint        string `env:"OBS_ENDPOINT"`
	Region          string `env:"OBS_REGION"`
	AccessKeyID     string `env:"OBS_ACCESS_KEY_ID"`
	AccessKeySecret string `env:"OBS_SECRET_ACCESS_KEY"`
	BucketName      string `env:"OBS_BUCKET"`
	ProxyDomain     string `env:"OBS_PROXY_DOMAIN"` // Optional proxy domain for public access
}

// NewObsStore creates an OBS storage instance from environment variables.
func NewObsStore() Store {
	return &ObsStore{
		Endpoint:        utils.GetEnv("OBS_ENDPOINT"),
		Region:          utils.GetEnv("OBS_REGION"),
		AccessKeyID:     utils.GetEnv("OBS_ACCESS_KEY_ID"),
		AccessKeySecret: utils.GetEnv("OBS_SECRET_ACCESS_KEY"),
		BucketName:      utils.GetEnv("OBS_BUCKET"),
		ProxyDomain:     strings.TrimSuffix(utils.GetEnv("OBS_PROXY_DOMAIN"), "/"),
	}
}

func (o *ObsStore) client(ctx context.Context) (*s3.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(o.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(o.AccessKeyID, o.AccessKeySecret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	endpoint := o.Endpoint
	return s3.NewFromConfig(cfg, func(opts *s3.Options) {
		opts.BaseEndpoint = aws.String(endpoint)
		opts.UsePathStyle = true
	}), nil
}

// Read reads a file from OBS.
func (o *ObsStore) Read(key string) (io.ReadCloser, int64, error) {
	ctx := context.Background()
	client, err := o.client(ctx)
	if err != nil {
		return nil, 0, err
	}

	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(o.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object from OBS: %w", err)
	}

	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return result.Body, size, nil
}

// Write writes a file to OBS.
func (o *ObsStore) Write(key string, r io.Reader) error {
	ctx := context.Background()
	client, err := o.client(ctx)
	if err != nil {
		return err
	}

	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(o.BucketName),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object to OBS: %w", err)
	}
	return nil
}

// Delete deletes a file from OBS.
func (o *ObsStore) Delete(key string) error {
	ctx := context.Background()
	client, err := o.client(ctx)
	if err != nil {
		return err
	}

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(o.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from OBS: %w", err)
	}
	return nil
}

// Exists checks if a file exists in OBS.
func (o *ObsStore) Exists(key string) (bool, error) {
	ctx := context.Background()
	client, err := o.client(ctx)
	if err != nil {
		return false, err
	}

	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(o.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence in OBS: %w", err)
	}
	return true, nil
}

// PublicURL returns the public URL for a file in OBS.
func (o *ObsStore) PublicURL(key string) string {
	key = strings.TrimPrefix(key, "/")
	if o.ProxyDomain != "" {
		return fmt.Sprintf("%s/%s", o.ProxyDomain, key)
	}
	endpoint := strings.TrimSuffix(o.Endpoint, "/")
	return fmt.Sprintf("%s/%s/%s", endpoint, o.BucketName, key)
}
