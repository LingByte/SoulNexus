package live

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestListCertificates(t *testing.T) {
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
	certificates, err := client.ListCertificates("lingecho", "qiniu.lingecho.com")
	assert.Nil(err)
	assert.NotNil(certificates)
	for _, certificate := range certificates {
		t.Log(certificate)
	}
}
