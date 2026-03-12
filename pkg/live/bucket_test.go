package live

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewBucketClient(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("not found QINIU_ACCESS_KEY or QINIU_SECRET_KEY")
	}
	client, err := NewBucketClient()
	if err != nil {
		t.Fatal(err)
	}
	assert := assert.New(t)
	assert.NotNil(client)
	assert.NotNil(client.httpClient)
	assert.NotNil(client.baseHost)
	assert.NotNil(client.accessKey)
	assert.NotNil(client.secretKey)
}

func TestCreateBucket(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("not found QINIU_ACCESS_KEY or QINIU_SECRET_KEY")
	}
	client, err := NewBucketClient()
	if err != nil {
		t.Fatal(err)
	}
	assert := assert.New(t)
	exists, _ := client.BucketExists("lingecho001")
	if exists {
		client.DeleteBucket("lingecho001")
	} else {
		bucket, err := client.CreateBucket("lingecho001")
		assert.Nil(err)
		assert.NotNil(bucket)
		assert.Equal("lingecho001", bucket.Name)
		assert.Equal("cn-east-1", bucket.Region)
		assert.Equal("Enabled", bucket.Status)
		t.Log(bucket)
		client.DeleteBucket("lingecho001")
	}
}

func TestDeleteBucket(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("not found QINIU_ACCESS_KEY or QINIU_SECRET_KEY")
	}
	client, err := NewBucketClient()
	if err != nil {
		t.Fatal(err)
	}
	assert := assert.New(t)
	bucket, err := client.DeleteBucket("lingecho001")
	assert.Nil(err)
	assert.NotNil(bucket)
	t.Log(bucket)
}

func TestListBuckets(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("not found QINIU_ACCESS_KEY or QINIU_SECRET_KEY")
	}
	client, err := NewBucketClient()
	if err != nil {
		t.Fatal(err)
	}
	assert := assert.New(t)
	buckets, err := client.ListBuckets()
	assert.Nil(err)
	assert.NotNil(buckets)
	t.Log(buckets)
}
