package text

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestQiniuGetTextCensor(t *testing.T) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skipf("not found QINIU_ACCESS_KEY or QINIU_SECRET_KEY")
	}
	textCensor, err := GetTextCensor(KindQiNiu)
	if err != nil {
		t.Error(err)
	}
	result, err := textCensor.CensorText("hello world")
	if err != nil {
		t.Error(err)
	}
	if result != nil {
		t.Logf("Suggestion: %s, Label: %s, Score: %.4f, Msg: %s", result.Suggestion, result.Label, result.Score, result.Msg)
	}
}

func TestQCloudGetTextCensor(t *testing.T) {
	secretID := utils.GetEnv("QCLOUD_SECRET_ID")
	secretKey := utils.GetEnv("QCLOUD_SECRET_KEY")
	if secretID == "" || secretKey == "" {
		t.Skipf("not found QCLOUD_SECRET_ID or QCLOUD_SECRET_KEY")
	}
	textCensor, err := GetTextCensor(KindQCloud)
	if err != nil {
		t.Error(err)
	}
	result, err := textCensor.CensorText("hello world")
	if err != nil {
		t.Error(err)
	}
	if result != nil {
		t.Logf("Suggestion: %s, Label: %s, Score: %.4f, Msg: %s", result.Suggestion, result.Label, result.Score, result.Msg)
	}
}

func TestAliyunGetTextCensor(t *testing.T) {
	accessKeyID := utils.GetEnv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := utils.GetEnv("ALIYUN_ACCESS_KEY_SECRET")
	if accessKeyID == "" || accessKeySecret == "" {
		t.Skipf("not found ALIYUN_ACCESS_KEY_ID or ALIYUN_ACCESS_KEY_SECRET")
	}
	textCensor, err := GetTextCensor(KindAliyun)
	if err != nil {
		t.Error(err)
	}
	result, err := textCensor.CensorText("hello world")
	if err != nil {
		t.Error(err)
	}
	if result != nil {
		t.Logf("Suggestion: %s, Label: %s, Score: %.4f, Msg: %s", result.Suggestion, result.Label, result.Score, result.Msg)
	}
}
