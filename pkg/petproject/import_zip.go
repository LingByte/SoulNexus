package petproject

import (
	"bytes"
	"fmt"
	"io"
	"path"
)

const StagingRoot = "pet-staging"

// StagingZipKey is the temporary object key for an uploaded .soulpet.zip.
func StagingZipKey(packageID string) string {
	return path.Join(StagingRoot, packageID, "package.zip")
}

// FullStore is object storage with read/write/delete used for zip import.
type FullStore interface {
	Writer
	Reader
	Deleter
}

// Deleter removes an object key.
type Deleter interface {
	Delete(key string) error
}

// PublishZip uploads zip to staging, extracts files under destPrefix, then deletes the staging zip.
func PublishZip(store FullStore, packageID string, zipData []byte, destPrefix string) (map[string]string, error) {
	if len(zipData) == 0 {
		return nil, fmt.Errorf("empty zip")
	}
	stagingKey := StagingZipKey(packageID)
	if err := store.Write(stagingKey, bytes.NewReader(zipData)); err != nil {
		return nil, fmt.Errorf("stage zip: %w", err)
	}
	defer func() { _ = store.Delete(stagingKey) }()

	rc, _, err := store.Read(stagingKey)
	if err != nil {
		return nil, fmt.Errorf("read staged zip: %w", err)
	}
	raw, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return nil, err
	}

	files, err := UnpackZip(raw)
	if err != nil {
		return nil, err
	}
	if err := WriteFiles(store, destPrefix, files); err != nil {
		return nil, err
	}
	return files, nil
}
