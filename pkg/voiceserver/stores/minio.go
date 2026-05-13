package stores

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStore struct {
	Endpoint  string `env:"MINIO_ENDPOINT"`
	AccessKey string `env:"MINIO_ACCESS_KEY"`
	SecretKey string `env:"MINIO_SECRET_KEY"`
	Bucket    string `env:"MINIO_BUCKET"`
	UseSSL    bool   `env:"MINIO_USE_SSL"`
	BaseURL   string `env:"MINIO_PUBLIC_BASE"`
}

func NewMinioStore() Store {
	useSSL := utils.GetEnv("MINIO_USE_SSL") == "1" || strings.ToLower(utils.GetEnv("MINIO_USE_SSL")) == "true"
	return &MinioStore{
		Endpoint:  utils.GetEnv("MINIO_ENDPOINT"),
		AccessKey: utils.GetEnv("MINIO_ACCESS_KEY"),
		SecretKey: utils.GetEnv("MINIO_SECRET_KEY"),
		Bucket:    utils.GetEnv("MINIO_BUCKET"),
		UseSSL:    useSSL,
		BaseURL:   utils.GetEnv("MINIO_PUBLIC_BASE"),
	}
}

func (m *MinioStore) client() (*minio.Client, error) {
	return minio.New(m.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(m.AccessKey, m.SecretKey, ""),
		Secure: m.UseSSL,
	})
}

func (m *MinioStore) ensureBucket(bucketName string, ctx context.Context, cli *minio.Client) error {
	if bucketName == "" {
		bucketName = m.Bucket
	}
	exists, err := cli.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !exists {
		return cli.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}
	return nil
}

func (m *MinioStore) Read(bucketName string, key string) (io.ReadCloser, int64, error) {
	if bucketName == "" {
		bucketName = m.Bucket
	}
	cli, err := m.client()
	if err != nil {
		return nil, 0, err
	}
	obj, err := cli.GetObject(context.Background(), bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	st, err := obj.Stat()
	if err != nil {
		return nil, 0, err
	}
	return obj, st.Size, nil
}

func (m *MinioStore) Write(bucketName string, key string, r io.Reader) error {
	if bucketName == "" {
		bucketName = m.Bucket
	}
	cli, err := m.client()
	if err != nil {
		return err
	}
	if err := m.ensureBucket(bucketName, context.Background(), cli); err != nil {
		return err
	}
	_, err = cli.PutObject(context.Background(), m.Bucket, key, r, -1, minio.PutObjectOptions{ContentType: http.DetectContentType([]byte{})})
	return err
}

func (m *MinioStore) Delete(bucketName string, key string) error {
	if bucketName == "" {
		bucketName = m.Bucket
	}
	cli, err := m.client()
	if err != nil {
		return err
	}
	return cli.RemoveObject(context.Background(), bucketName, key, minio.RemoveObjectOptions{})
}

func (m *MinioStore) Exists(bucketName string, key string) (bool, error) {
	if bucketName == "" {
		bucketName = m.Bucket
	}
	cli, err := m.client()
	if err != nil {
		return false, err
	}
	_, err = cli.StatObject(context.Background(), bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *MinioStore) PublicURL(bucketName string, key string) string {
	if bucketName == "" {
		bucketName = m.Bucket
	}

	// 如果设置了自定义域名，使用自定义域名
	if m.BaseURL != "" {
		return strings.TrimRight(m.BaseURL, "/") + "/" + bucketName + "/" + key
	}

	// 使用默认的MinIO域名格式
	scheme := "http://"
	if m.UseSSL {
		scheme = "https://"
	}
	return scheme + m.Endpoint + "/" + bucketName + "/" + key
}

func (m *MinioStore) SetBaseURL(baseURL string) {
	m.BaseURL = baseURL
}
