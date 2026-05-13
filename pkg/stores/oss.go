package stores

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSStore struct {
	AccessKeyID     string `env:"OSS_ACCESS_KEY_ID"`
	AccessKeySecret string `env:"OSS_ACCESS_KEY_SECRET"`
	Endpoint        string `env:"OSS_ENDPOINT"`
	BucketName      string `env:"OSS_BUCKET_NAME"`
	baseURL         string
}

// Delete implements Store.
func (o *OSSStore) Delete(key string) error {
	if o.AccessKeyID == "" || o.AccessKeySecret == "" || o.Endpoint == "" {
		return fmt.Errorf("OSS credentials not configured")
	}

	client, err := oss.New(o.Endpoint, o.AccessKeyID, o.AccessKeySecret)
	if err != nil {
		return fmt.Errorf("failed to create OSS client: %v", err)
	}
	
	bucket, err := client.Bucket(o.BucketName)
	if err != nil {
		return fmt.Errorf("failed to get bucket: %v", err)
	}

	err = bucket.DeleteObject(key)
	return err
}

// Exists implements Store.
func (o *OSSStore) Exists(key string) (bool, error) {
	if o.AccessKeyID == "" || o.AccessKeySecret == "" || o.Endpoint == "" {
		return false, fmt.Errorf("OSS credentials not configured")
	}

	client, err := oss.New(o.Endpoint, o.AccessKeyID, o.AccessKeySecret)
	if err != nil {
		return false, fmt.Errorf("failed to create OSS client: %v", err)
	}

	bucket, err := client.Bucket(o.BucketName)
	if err != nil {
		return false, fmt.Errorf("failed to get bucket: %v", err)
	}

	ok, err := bucket.IsObjectExist(key)
	return ok, err
}

// Read implements Store.
func (o *OSSStore) Read(key string) (io.ReadCloser, int64, error) {
	if o.AccessKeyID == "" || o.AccessKeySecret == "" || o.Endpoint == "" {
		return nil, 0, fmt.Errorf("OSS credentials not configured")
	}

	client, err := oss.New(o.Endpoint, o.AccessKeyID, o.AccessKeySecret)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create OSS client: %v", err)
	}

	bucket, err := client.Bucket(o.BucketName)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get bucket: %v", err)
	}

	// 先获取对象信息以获取大小
	props, err := bucket.GetObjectDetailedMeta(key)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object meta: %v", err)
	}

	var size int64 = -1
	if cl := props.Get("Content-Length"); cl != "" {
		if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
			size = v
		}
	}

	// 获取对象内容
	body, err := bucket.GetObject(key)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object: %v", err)
	}

	return body, size, nil
}

// Write implements Store.
func (o *OSSStore) Write(key string, r io.Reader) error {
	if o.AccessKeyID == "" || o.AccessKeySecret == "" || o.Endpoint == "" {
		return fmt.Errorf("OSS credentials not configured")
	}

	client, err := oss.New(o.Endpoint, o.AccessKeyID, o.AccessKeySecret)
	if err != nil {
		return fmt.Errorf("failed to create OSS client: %v", err)
	}

	bucket, err := client.Bucket(o.BucketName)
	if err != nil {
		return fmt.Errorf("failed to get bucket: %v", err)
	}

	err = bucket.PutObject(key, r)
	return err
}

func (o *OSSStore) PublicURL(key string) string {
	if o.baseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(o.baseURL, "/"), key)
	}
	endpoint := strings.TrimPrefix(o.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return fmt.Sprintf("https://%s.%s/%s", o.BucketName, endpoint, key)
}

func NewOSSStore() Store {
	return &OSSStore{
		AccessKeyID:     utils.GetEnv("OSS_ACCESS_KEY_ID"),
		AccessKeySecret: utils.GetEnv("OSS_ACCESS_KEY_SECRET"),
		Endpoint:        utils.GetEnv("OSS_ENDPOINT"),
		BucketName:      utils.GetEnv("OSS_BUCKET_NAME"),
	}
}

func (o *OSSStore) SetBaseURL(baseURL string) {
	o.baseURL = baseURL
}
