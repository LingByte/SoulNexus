package stores

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type CosStore struct {
	SecretID   string `env:"COS_SECRET_ID"`
	SecretKey  string `env:"COS_SECRET_KEY"`
	Region     string `env:"COS_REGION"`
	BucketName string `env:"COS_BUCKET_NAME"`
}

// Delete implements Store.
func (c *CosStore) Delete(bucketName string, key string) error {
	if c.SecretID == "" || c.SecretKey == "" || c.Region == "" {
		return fmt.Errorf("COS credentials not configured")
	}

	cClient := InitCos(bucketName, c)
	_, err := cClient.Object.Delete(context.Background(), key)
	return err
}

// Exists implements Store.
func (c *CosStore) Exists(bucketName string, key string) (bool, error) {
	if c.SecretID == "" || c.SecretKey == "" || c.Region == "" {
		return false, fmt.Errorf("COS credentials not configured")
	}

	cClient := InitCos(bucketName, c)
	ok, err := cClient.Object.IsExist(context.Background(), key)
	return ok, err
}

// Read implements Store.
func (c *CosStore) Read(bucketName string, key string) (io.ReadCloser, int64, error) {
	if c.SecretID == "" || c.SecretKey == "" || c.Region == "" {
		return nil, 0, fmt.Errorf("COS credentials not configured")
	}

	cClient := InitCos(bucketName, c)

	// 直接获取对象
	resp, err := cClient.Object.Get(context.Background(), key, nil)
	if err != nil {
		return nil, 0, err
	}

	// 获取内容长度
	var size int64 = -1
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if v, err := fmt.Sscanf(cl, "%d", &size); err != nil || v != 1 {
			size = -1
		}
	}

	return resp.Body, size, nil
}

// Write implements Store.
func (c *CosStore) Write(bucketName string, key string, r io.Reader) error {
	if c.SecretID == "" || c.SecretKey == "" || c.Region == "" {
		return fmt.Errorf("COS credentials not configured")
	}

	cClient := InitCos(bucketName, c)
	_, err := cClient.Object.Put(context.Background(), key, r, nil)
	return err
}

func (c *CosStore) PublicURL(bucketName string, key string) string {
	if bucketName == "" {
		bucketName = c.BucketName
	}
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", bucketName, c.Region, key)
}

func NewCosStore() Store {
	return &CosStore{
		SecretID:   utils.GetEnv("COS_SECRET_ID"),
		SecretKey:  utils.GetEnv("COS_SECRET_KEY"),
		Region:     utils.GetEnv("COS_REGION"),
		BucketName: utils.GetEnv("COS_BUCKET_NAME"),
	}
}

func InitCos(bucketName string, c *CosStore) *cos.Client {
	if bucketName == "" {
		bucketName = c.BucketName
	}
	bucketURL := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucketName, c.Region)
	u, _ := url.Parse(bucketURL)
	b := &cos.BaseURL{BucketURL: u}

	cClient := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.SecretID,
			SecretKey: c.SecretKey,
		},
	})
	return cClient
}
