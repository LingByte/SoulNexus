// Package stores provides a unified storage abstraction layer supporting
// multiple cloud object storage backends and local file system storage.
//
// All backends implement the Store interface with five operations:
// Read, Write, Delete, Exists, and PublicURL. Configuration is injected
// through environment variables following the twelve-factor app pattern.
//
// Supported for:
//
//	local   - Local filesystem (default fallback)
//	cos     - Tencent Cloud COS
//	minio   - MinIO / S3 compatible
//	qiniu   - Qiniu Kodo
//	oss     - Alibaba Cloud OSS
//	s3      - AWS S3 / S3 compatible
//	tos     - Volcengine TOS
//	obs     - Huawei Cloud OBS
//	ks3     - Kingsoft Cloud KS3
//
// Usage:
//
//	// Use default store (from STORAGE_KIND env var)
//	store := stores.Default()
//	r, size, err := store.Read("some-key")
//
//	// Or get a specific backend
//	store := stores.GetStore(stores.KindS3)
package stores
