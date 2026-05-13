package stores

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type QiNiuStore struct {
	AccessKey  string `env:"QINIU_ACCESS_KEY"`
	SecretKey  string `env:"QINIU_SECRET_KEY"`
	BucketName string `env:"QINIU_BUCKET"`
	Domain     string `env:"QINIU_DOMAIN"`
	Private    bool   `env:"QINIU_PRIVATE"`
	Region     string `env:"QINIU_REGION"`
}

func NewQiNiuStore() Store {
	private := strings.EqualFold(utils.GetEnv("QINIU_PRIVATE"), "true")
	return &QiNiuStore{
		AccessKey:  utils.GetEnv("QINIU_ACCESS_KEY"),
		SecretKey:  utils.GetEnv("QINIU_SECRET_KEY"),
		BucketName: utils.GetEnv("QINIU_BUCKET"),
		Domain:     utils.GetEnv("QINIU_DOMAIN"),
		Private:    private,
		Region:     utils.GetEnv("QINIU_REGION"),
	}
}

func (q *QiNiuStore) getMac() *qbox.Mac {
	return qbox.NewMac(q.AccessKey, q.SecretKey)
}

func (q *QiNiuStore) makeConfig(bucketName string) storage.Config {
	if bucketName == "" {
		bucketName = q.BucketName
	}
	useHTTPS := strings.HasPrefix(strings.ToLower(q.Domain), "https://")
	cfg := storage.Config{
		UseHTTPS: useHTTPS,
	}
	if zone, err := storage.GetRegion(q.AccessKey, bucketName); err == nil && zone != nil {
		cfg.Region = zone
	}
	return cfg
}

func (q *QiNiuStore) uploadToken(bucketName string) string {
	if bucketName == "" {
		bucketName = q.BucketName
	}
	p := storage.PutPolicy{
		Scope:   bucketName,
		Expires: 3600, // 1小时
	}
	return p.UploadToken(q.getMac())
}

// Write: 使用表单上传（将 r 读入内存以得到内容长度，适合中小文件；大文件建议换分片上传）
func (q *QiNiuStore) Write(key string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	cfg := q.makeConfig(q.BucketName)
	uploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	extra := storage.PutExtra{}
	token := q.uploadToken(q.BucketName)
	ctx := context.Background()
	return uploader.Put(ctx, &ret, token, key, bytes.NewReader(data), int64(len(data)), &extra)
}

// Exists: 通过 Stat 判断（612 表示不存在）
func (q *QiNiuStore) Exists(key string) (bool, error) {

	cfg := q.makeConfig(q.BucketName)
	bm := storage.NewBucketManager(q.getMac(), &cfg)
	_, err := bm.Stat(q.BucketName, key)
	if err == nil {
		return true, nil
	}
	if e, ok := err.(*storage.ErrorInfo); ok && e.Code == 612 {
		return false, nil
	}
	return false, err
}

// Delete: 直接删除
func (q *QiNiuStore) Delete(key string) error {
	cfg := q.makeConfig(q.BucketName)
	bm := storage.NewBucketManager(q.getMac(), &cfg)
	return bm.Delete(q.BucketName, key)
}

// Read: 通过 PublicURL（公有或带签名的私有）发起 HTTP GET
func (q *QiNiuStore) Read(key string) (io.ReadCloser, int64, error) {
	u := q.PublicURL(key)
	if u == "" {
		return nil, 0, ErrInvalidPath
	}
	resp, err := http.Get(u)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, 0, &utils.Error{Code: resp.StatusCode, Message: "qiniu read failed"}
	}
	var n int64 = -1
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
			n = v
		}
	}
	return resp.Body, n, nil
}

// PublicURL: 公有空间返回公开 URL；私有空间返回带有效期签名的 URL（默认 1 小时）
func (q *QiNiuStore) PublicURL(key string) string {
	if q.Domain == "" {
		return ""
	}
	d := q.Domain
	if !strings.HasPrefix(d, "http://") && !strings.HasPrefix(d, "https://") {
		d = "http://" + d
	}
	// 公有 URL
	pub := storage.MakePublicURLv2(d, key)

	if !q.Private {
		return pub
	}
	// 私有下载 URL（签名，有效期 1 小时）
	deadline := time.Now().Add(1 * time.Hour).Unix()
	return storage.MakePrivateURL(q.getMac(), d, key, deadline)
}
