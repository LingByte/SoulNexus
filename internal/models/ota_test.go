package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupOTATestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t, &OTA{})
}

func TestOTA_TableName(t *testing.T) {
	var ota OTA
	assert.Equal(t, "ai_ota", ota.TableName())
}

func TestCreateOTA(t *testing.T) {
	db := setupOTATestDB(t)

	ota := &OTA{
		ID:           "ota-001",
		FirmwareName: "firmware-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		Size:         1024000,
		Remark:       "Initial release",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
		Sort:         1,
	}

	err := CreateOTA(db, ota)
	require.NoError(t, err)

	// Verify the record was created
	var retrieved OTA
	err = db.First(&retrieved, "id = ?", "ota-001").Error
	require.NoError(t, err)
	assert.Equal(t, "ota-001", retrieved.ID)
	assert.Equal(t, "firmware-v1.0.0", retrieved.FirmwareName)
	assert.Equal(t, "esp32", retrieved.Type)
	assert.Equal(t, "1.0.0", retrieved.Version)
	assert.Equal(t, int64(1024000), retrieved.Size)
	assert.Equal(t, "Initial release", retrieved.Remark)
	assert.Equal(t, "/firmware/esp32/v1.0.0.bin", retrieved.FirmwarePath)
	assert.Equal(t, 1, retrieved.Sort)
	assert.False(t, retrieved.CreatedAt.IsZero())
	assert.False(t, retrieved.UpdatedAt.IsZero())
}

func TestCreateOTA_WithMinimalFields(t *testing.T) {
	db := setupOTATestDB(t)

	ota := &OTA{
		ID:           "ota-002",
		FirmwareName: "firmware-v2.0.0",
		Type:         "default",
		Version:      "2.0.0",
		FirmwarePath: "/firmware/default/v2.0.0.bin",
	}

	err := CreateOTA(db, ota)
	require.NoError(t, err)

	// Verify default values
	var retrieved OTA
	err = db.First(&retrieved, "id = ?", "ota-002").Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), retrieved.Size)
	assert.Equal(t, "", retrieved.Remark)
	assert.Equal(t, 0, retrieved.Sort)
}

func TestGetLatestOTA_Success(t *testing.T) {
	db := setupOTATestDB(t)

	// Create multiple OTA records for the same type
	ota1 := &OTA{
		ID:           "ota-001",
		FirmwareName: "firmware-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
	}
	err := CreateOTA(db, ota1)
	require.NoError(t, err)

	// Wait a bit to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	ota2 := &OTA{
		ID:           "ota-002",
		FirmwareName: "firmware-v1.1.0",
		Type:         "esp32",
		Version:      "1.1.0",
		FirmwarePath: "/firmware/esp32/v1.1.0.bin",
	}
	err = CreateOTA(db, ota2)
	require.NoError(t, err)

	// Wait a bit more
	time.Sleep(10 * time.Millisecond)

	ota3 := &OTA{
		ID:           "ota-003",
		FirmwareName: "firmware-v1.2.0",
		Type:         "esp32",
		Version:      "1.2.0",
		FirmwarePath: "/firmware/esp32/v1.2.0.bin",
	}
	err = CreateOTA(db, ota3)
	require.NoError(t, err)

	// Get latest OTA for esp32 type
	latest, err := GetLatestOTA(db, "esp32")
	require.NoError(t, err)
	assert.Equal(t, "ota-003", latest.ID)
	assert.Equal(t, "firmware-v1.2.0", latest.FirmwareName)
	assert.Equal(t, "1.2.0", latest.Version)
}

func TestGetLatestOTA_NotFound(t *testing.T) {
	db := setupOTATestDB(t)

	// Try to get OTA for non-existent type
	_, err := GetLatestOTA(db, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestGetLatestOTA_DifferentTypes(t *testing.T) {
	db := setupOTATestDB(t)

	// Create OTA records for different types
	ota1 := &OTA{
		ID:           "ota-esp32-001",
		FirmwareName: "firmware-esp32-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
	}
	err := CreateOTA(db, ota1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	ota2 := &OTA{
		ID:           "ota-default-001",
		FirmwareName: "firmware-default-v1.0.0",
		Type:         "default",
		Version:      "1.0.0",
		FirmwarePath: "/firmware/default/v1.0.0.bin",
	}
	err = CreateOTA(db, ota2)
	require.NoError(t, err)

	// Get latest for each type
	latestESP32, err := GetLatestOTA(db, "esp32")
	require.NoError(t, err)
	assert.Equal(t, "ota-esp32-001", latestESP32.ID)
	assert.Equal(t, "esp32", latestESP32.Type)

	latestDefault, err := GetLatestOTA(db, "default")
	require.NoError(t, err)
	assert.Equal(t, "ota-default-001", latestDefault.ID)
	assert.Equal(t, "default", latestDefault.Type)
}

func TestUpdateOTA(t *testing.T) {
	db := setupOTATestDB(t)

	// Create an OTA record
	ota := &OTA{
		ID:           "ota-001",
		FirmwareName: "firmware-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		Size:         1024000,
		Remark:       "Initial release",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
		Sort:         1,
	}
	err := CreateOTA(db, ota)
	require.NoError(t, err)

	originalUpdatedAt := ota.UpdatedAt

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Update the OTA record
	ota.FirmwareName = "firmware-v1.0.1"
	ota.Version = "1.0.1"
	ota.Size = 2048000
	ota.Remark = "Bug fixes"
	ota.Sort = 2

	err = UpdateOTA(db, ota)
	require.NoError(t, err)

	// Verify the update
	var retrieved OTA
	err = db.First(&retrieved, "id = ?", "ota-001").Error
	require.NoError(t, err)
	assert.Equal(t, "firmware-v1.0.1", retrieved.FirmwareName)
	assert.Equal(t, "1.0.1", retrieved.Version)
	assert.Equal(t, int64(2048000), retrieved.Size)
	assert.Equal(t, "Bug fixes", retrieved.Remark)
	assert.Equal(t, 2, retrieved.Sort)
	assert.True(t, retrieved.UpdatedAt.After(originalUpdatedAt))
}

func TestUpdateOTA_AllFields(t *testing.T) {
	db := setupOTATestDB(t)

	// Create an OTA record
	ota := &OTA{
		ID:           "ota-001",
		FirmwareName: "firmware-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
	}
	err := CreateOTA(db, ota)
	require.NoError(t, err)

	// Update all fields
	ota.FirmwareName = "firmware-v2.0.0"
	ota.Type = "default"
	ota.Version = "2.0.0"
	ota.Size = 3072000
	ota.Remark = "Major update"
	ota.FirmwarePath = "/firmware/default/v2.0.0.bin"
	ota.Sort = 10

	err = UpdateOTA(db, ota)
	require.NoError(t, err)

	// Verify all fields were updated
	var retrieved OTA
	err = db.First(&retrieved, "id = ?", "ota-001").Error
	require.NoError(t, err)
	assert.Equal(t, "firmware-v2.0.0", retrieved.FirmwareName)
	assert.Equal(t, "default", retrieved.Type)
	assert.Equal(t, "2.0.0", retrieved.Version)
	assert.Equal(t, int64(3072000), retrieved.Size)
	assert.Equal(t, "Major update", retrieved.Remark)
	assert.Equal(t, "/firmware/default/v2.0.0.bin", retrieved.FirmwarePath)
	assert.Equal(t, 10, retrieved.Sort)
}

func TestDeleteOTA_Success(t *testing.T) {
	db := setupOTATestDB(t)

	// Create an OTA record
	ota := &OTA{
		ID:           "ota-001",
		FirmwareName: "firmware-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
	}
	err := CreateOTA(db, ota)
	require.NoError(t, err)

	// Verify it exists
	var count int64
	db.Model(&OTA{}).Where("id = ?", "ota-001").Count(&count)
	assert.Equal(t, int64(1), count)

	// Delete the OTA record
	err = DeleteOTA(db, "ota-001")
	require.NoError(t, err)

	// Verify it was deleted
	db.Model(&OTA{}).Where("id = ?", "ota-001").Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeleteOTA_NonExistent(t *testing.T) {
	db := setupOTATestDB(t)

	// Try to delete non-existent OTA
	err := DeleteOTA(db, "nonexistent-id")
	// Delete should not error even if record doesn't exist
	// GORM's Delete returns no error if no rows are affected
	assert.NoError(t, err)
}

func TestDeleteOTA_MultipleRecords(t *testing.T) {
	db := setupOTATestDB(t)

	// Create multiple OTA records
	ota1 := &OTA{
		ID:           "ota-001",
		FirmwareName: "firmware-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
	}
	err := CreateOTA(db, ota1)
	require.NoError(t, err)

	ota2 := &OTA{
		ID:           "ota-002",
		FirmwareName: "firmware-v1.1.0",
		Type:         "esp32",
		Version:      "1.1.0",
		FirmwarePath: "/firmware/esp32/v1.1.0.bin",
	}
	err = CreateOTA(db, ota2)
	require.NoError(t, err)

	// Verify both exist
	var count int64
	db.Model(&OTA{}).Count(&count)
	assert.Equal(t, int64(2), count)

	// Delete one
	err = DeleteOTA(db, "ota-001")
	require.NoError(t, err)

	// Verify only one remains
	db.Model(&OTA{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// Verify the correct one was deleted
	var remaining OTA
	err = db.First(&remaining, "id = ?", "ota-002").Error
	require.NoError(t, err)
	assert.Equal(t, "ota-002", remaining.ID)
}

func TestGetLatestOTA_OrderByUpdatedAt(t *testing.T) {
	db := setupOTATestDB(t)

	// Create first OTA
	ota1 := &OTA{
		ID:           "ota-001",
		FirmwareName: "firmware-v1.0.0",
		Type:         "esp32",
		Version:      "1.0.0",
		FirmwarePath: "/firmware/esp32/v1.0.0.bin",
	}
	err := CreateOTA(db, ota1)
	require.NoError(t, err)

	// Wait to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Create second OTA
	ota2 := &OTA{
		ID:           "ota-002",
		FirmwareName: "firmware-v1.1.0",
		Type:         "esp32",
		Version:      "1.1.0",
		FirmwarePath: "/firmware/esp32/v1.1.0.bin",
	}
	err = CreateOTA(db, ota2)
	require.NoError(t, err)

	// Wait and update the first one to make it newer
	time.Sleep(10 * time.Millisecond)
	ota1.Remark = "Updated"
	err = UpdateOTA(db, ota1)
	require.NoError(t, err)

	// Get latest - should return ota-001 because it was updated most recently
	latest, err := GetLatestOTA(db, "esp32")
	require.NoError(t, err)
	assert.Equal(t, "ota-001", latest.ID)
	assert.Equal(t, "Updated", latest.Remark)
}

func TestOTA_WithEmptyFields(t *testing.T) {
	db := setupOTATestDB(t)

	// Create OTA with empty optional fields
	ota := &OTA{
		ID:           "ota-empty",
		FirmwareName: "",
		Type:         "",
		Version:      "",
		FirmwarePath: "",
	}
	err := CreateOTA(db, ota)
	require.NoError(t, err)

	// Verify it was created
	var retrieved OTA
	err = db.First(&retrieved, "id = ?", "ota-empty").Error
	require.NoError(t, err)
	assert.Equal(t, "ota-empty", retrieved.ID)
}

func TestGetLatestOTA_EmptyDatabase(t *testing.T) {
	db := setupOTATestDB(t)

	// Try to get OTA from empty database
	_, err := GetLatestOTA(db, "esp32")
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}
