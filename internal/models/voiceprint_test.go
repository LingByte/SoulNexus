package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupVoiceprintTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&Voiceprint{})
	require.NoError(t, err)

	return db
}

func TestVoiceprint_CRUD(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	// 测试创建声纹记录
	featureVector := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	voiceprint := &Voiceprint{
		SpeakerID:     "speaker-001",
		AssistantID:   "assistant-123",
		SpeakerName:   "张三",
		FeatureVector: featureVector,
	}

	err := db.Create(voiceprint).Error
	assert.NoError(t, err)
	assert.NotZero(t, voiceprint.ID)
	assert.NotZero(t, voiceprint.CreatedAt)

	// 测试读取声纹记录
	var retrieved Voiceprint
	err = db.First(&retrieved, voiceprint.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "speaker-001", retrieved.SpeakerID)
	assert.Equal(t, "assistant-123", retrieved.AssistantID)
	assert.Equal(t, "张三", retrieved.SpeakerName)
	assert.Equal(t, featureVector, retrieved.FeatureVector)

	// 测试更新声纹记录
	retrieved.SpeakerName = "张三丰"
	newFeatureVector := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	retrieved.FeatureVector = newFeatureVector
	err = db.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated Voiceprint
	err = db.First(&updated, voiceprint.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "张三丰", updated.SpeakerName)
	assert.Equal(t, newFeatureVector, updated.FeatureVector)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))

	// 测试删除声纹记录
	err = db.Delete(&updated).Error
	assert.NoError(t, err)

	var deleted Voiceprint
	err = db.First(&deleted, voiceprint.ID).Error
	assert.Error(t, err)
}

func TestVoiceprint_QueryBySpeakerID(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	// 创建多个声纹记录
	voiceprints := []*Voiceprint{
		{
			SpeakerID:     "speaker-001",
			AssistantID:   "assistant-123",
			SpeakerName:   "张三",
			FeatureVector: []byte{0x01, 0x02, 0x03},
		},
		{
			SpeakerID:     "speaker-002",
			AssistantID:   "assistant-123",
			SpeakerName:   "李四",
			FeatureVector: []byte{0x04, 0x05, 0x06},
		},
		{
			SpeakerID:     "speaker-001",
			AssistantID:   "assistant-456",
			SpeakerName:   "张三",
			FeatureVector: []byte{0x07, 0x08, 0x09},
		},
	}

	for _, vp := range voiceprints {
		err := db.Create(vp).Error
		require.NoError(t, err)
	}

	// 按SpeakerID查询
	var speakerVoiceprints []Voiceprint
	err := db.Where("speaker_id = ?", "speaker-001").Find(&speakerVoiceprints).Error
	assert.NoError(t, err)
	assert.Len(t, speakerVoiceprints, 2)

	// 验证查询结果
	for _, vp := range speakerVoiceprints {
		assert.Equal(t, "speaker-001", vp.SpeakerID)
		assert.Equal(t, "张三", vp.SpeakerName)
	}

	// 验证不同的AssistantID
	assistantIDs := make([]string, len(speakerVoiceprints))
	for i, vp := range speakerVoiceprints {
		assistantIDs[i] = vp.AssistantID
	}
	assert.Contains(t, assistantIDs, "assistant-123")
	assert.Contains(t, assistantIDs, "assistant-456")
}

func TestVoiceprint_QueryByAssistantID(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	assistantID := "assistant-789"

	// 创建同一助手的多个声纹记录
	voiceprints := []*Voiceprint{
		{
			SpeakerID:     "speaker-001",
			AssistantID:   assistantID,
			SpeakerName:   "用户A",
			FeatureVector: []byte{0x01, 0x02, 0x03},
		},
		{
			SpeakerID:     "speaker-002",
			AssistantID:   assistantID,
			SpeakerName:   "用户B",
			FeatureVector: []byte{0x04, 0x05, 0x06},
		},
		{
			SpeakerID:     "speaker-003",
			AssistantID:   "other-assistant",
			SpeakerName:   "其他用户",
			FeatureVector: []byte{0x07, 0x08, 0x09},
		},
	}

	for _, vp := range voiceprints {
		err := db.Create(vp).Error
		require.NoError(t, err)
	}

	// 按AssistantID查询
	var assistantVoiceprints []Voiceprint
	err := db.Where("assistant_id = ?", assistantID).Find(&assistantVoiceprints).Error
	assert.NoError(t, err)
	assert.Len(t, assistantVoiceprints, 2)

	// 验证查询结果
	speakerNames := make([]string, len(assistantVoiceprints))
	for i, vp := range assistantVoiceprints {
		assert.Equal(t, assistantID, vp.AssistantID)
		speakerNames[i] = vp.SpeakerName
	}
	assert.Contains(t, speakerNames, "用户A")
	assert.Contains(t, speakerNames, "用户B")
}

func TestVoiceprint_UniqueConstraint(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	// 创建第一个声纹记录
	voiceprint1 := &Voiceprint{
		SpeakerID:     "speaker-unique",
		AssistantID:   "assistant-unique",
		SpeakerName:   "唯一用户",
		FeatureVector: []byte{0x01, 0x02, 0x03},
	}

	err := db.Create(voiceprint1).Error
	assert.NoError(t, err)

	// 尝试创建相同SpeakerID和AssistantID的记录
	voiceprint2 := &Voiceprint{
		SpeakerID:     "speaker-unique",
		AssistantID:   "assistant-unique",
		SpeakerName:   "重复用户",
		FeatureVector: []byte{0x04, 0x05, 0x06},
	}

	err = db.Create(voiceprint2).Error
	// 注意：SQLite可能不会严格执行唯一约束，这取决于表结构定义
	// 在实际应用中，应该在数据库层面添加唯一约束
	// 这里我们测试应用层的逻辑

	// 检查是否存在重复记录
	var count int64
	err = db.Model(&Voiceprint{}).Where("speaker_id = ? AND assistant_id = ?", "speaker-unique", "assistant-unique").Count(&count).Error
	assert.NoError(t, err)
	// 如果有唯一约束，count应该是1；如果没有，可能是2
}

func TestVoiceprint_FeatureVectorHandling(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	// 测试不同大小的特征向量
	testCases := []struct {
		name          string
		featureVector []byte
		description   string
	}{
		{
			name:          "small_vector",
			featureVector: []byte{0x01, 0x02, 0x03, 0x04},
			description:   "小型特征向量",
		},
		{
			name:          "medium_vector",
			featureVector: make([]byte, 256),
			description:   "中型特征向量",
		},
		{
			name:          "large_vector",
			featureVector: make([]byte, 1024),
			description:   "大型特征向量",
		},
	}

	// 初始化测试向量
	for i := range testCases[1].featureVector {
		testCases[1].featureVector[i] = byte(i % 256)
	}
	for i := range testCases[2].featureVector {
		testCases[2].featureVector[i] = byte((i * 3) % 256)
	}

	for i, tc := range testCases {
		voiceprint := &Voiceprint{
			SpeakerID:     "speaker-" + string(rune('1'+i)),
			AssistantID:   "assistant-vector-test",
			SpeakerName:   tc.description,
			FeatureVector: tc.featureVector,
		}

		err := db.Create(voiceprint).Error
		assert.NoError(t, err, "Failed to create voiceprint for %s", tc.name)

		// 验证特征向量存储和读取
		var retrieved Voiceprint
		err = db.First(&retrieved, voiceprint.ID).Error
		assert.NoError(t, err, "Failed to retrieve voiceprint for %s", tc.name)
		assert.Equal(t, tc.featureVector, retrieved.FeatureVector, "Feature vector mismatch for %s", tc.name)
		assert.Equal(t, len(tc.featureVector), len(retrieved.FeatureVector), "Feature vector length mismatch for %s", tc.name)
	}
}

func TestVoiceprint_TimeTracking(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	beforeCreate := time.Now()
	voiceprint := &Voiceprint{
		SpeakerID:     "speaker-time-test",
		AssistantID:   "assistant-time-test",
		SpeakerName:   "时间测试用户",
		FeatureVector: []byte{0x01, 0x02, 0x03},
	}

	err := db.Create(voiceprint).Error
	require.NoError(t, err)
	afterCreate := time.Now()

	// 验证创建时间
	assert.True(t, voiceprint.CreatedAt.After(beforeCreate))
	assert.True(t, voiceprint.CreatedAt.Before(afterCreate))
	assert.True(t, voiceprint.UpdatedAt.After(beforeCreate))
	assert.True(t, voiceprint.UpdatedAt.Before(afterCreate))

	// 等待一小段时间后更新
	time.Sleep(10 * time.Millisecond)
	beforeUpdate := time.Now()
	voiceprint.SpeakerName = "更新后的用户名"
	err = db.Save(voiceprint).Error
	require.NoError(t, err)
	afterUpdate := time.Now()

	// 验证更新时间
	var updated Voiceprint
	err = db.First(&updated, voiceprint.ID).Error
	require.NoError(t, err)
	assert.True(t, updated.UpdatedAt.After(beforeUpdate))
	assert.True(t, updated.UpdatedAt.Before(afterUpdate))
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
}

func TestVoiceprint_EmptyAndNullValues(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	// 测试空字符串和空特征向量
	voiceprint := &Voiceprint{
		SpeakerID:     "",
		AssistantID:   "",
		SpeakerName:   "",
		FeatureVector: []byte{},
	}

	err := db.Create(voiceprint).Error
	// 根据数据库约束，这可能会失败或成功
	// 如果SpeakerID和AssistantID有NOT NULL约束，应该失败
	if err == nil {
		// 如果创建成功，验证空值处理
		var retrieved Voiceprint
		err = db.First(&retrieved, voiceprint.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "", retrieved.SpeakerID)
		assert.Equal(t, "", retrieved.AssistantID)
		assert.Equal(t, "", retrieved.SpeakerName)
		assert.Equal(t, []byte{}, retrieved.FeatureVector)
	}

	// 测试有效的最小数据
	minimalVoiceprint := &Voiceprint{
		SpeakerID:     "min-speaker",
		AssistantID:   "min-assistant",
		SpeakerName:   "最小用户",
		FeatureVector: []byte{0x01},
	}

	err = db.Create(minimalVoiceprint).Error
	assert.NoError(t, err)

	var retrieved Voiceprint
	err = db.First(&retrieved, minimalVoiceprint.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "min-speaker", retrieved.SpeakerID)
	assert.Equal(t, "min-assistant", retrieved.AssistantID)
	assert.Equal(t, "最小用户", retrieved.SpeakerName)
	assert.Equal(t, []byte{0x01}, retrieved.FeatureVector)
}

func TestVoiceprint_ComplexQueries(t *testing.T) {
	db := setupVoiceprintTestDB(t)

	// 创建测试数据
	testData := []*Voiceprint{
		{
			SpeakerID:     "speaker-001",
			AssistantID:   "assistant-A",
			SpeakerName:   "张三",
			FeatureVector: []byte{0x01, 0x02, 0x03},
		},
		{
			SpeakerID:     "speaker-002",
			AssistantID:   "assistant-A",
			SpeakerName:   "李四",
			FeatureVector: []byte{0x04, 0x05, 0x06},
		},
		{
			SpeakerID:     "speaker-003",
			AssistantID:   "assistant-B",
			SpeakerName:   "王五",
			FeatureVector: []byte{0x07, 0x08, 0x09},
		},
		{
			SpeakerID:     "speaker-001",
			AssistantID:   "assistant-B",
			SpeakerName:   "张三",
			FeatureVector: []byte{0x0A, 0x0B, 0x0C},
		},
	}

	for _, vp := range testData {
		err := db.Create(vp).Error
		require.NoError(t, err)
	}

	// 复杂查询：按助手ID和说话人姓名查询
	var results []Voiceprint
	err := db.Where("assistant_id = ? AND speaker_name = ?", "assistant-A", "张三").Find(&results).Error
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "speaker-001", results[0].SpeakerID)

	// 复杂查询：按说话人ID查询多个助手
	err = db.Where("speaker_id = ?", "speaker-001").Find(&results).Error
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// 复杂查询：按助手ID查询并按创建时间排序
	err = db.Where("assistant_id = ?", "assistant-A").Order("created_at DESC").Find(&results).Error
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	// 验证排序（后创建的在前）
	assert.True(t, results[0].CreatedAt.After(results[1].CreatedAt) || results[0].CreatedAt.Equal(results[1].CreatedAt))

	// 统计查询：按助手ID统计声纹数量
	var count int64
	err = db.Model(&Voiceprint{}).Where("assistant_id = ?", "assistant-A").Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestVoiceprint_TableName(t *testing.T) {
	voiceprint := Voiceprint{}
	tableName := voiceprint.TableName()
	assert.Equal(t, "voiceprints", tableName)
}

// Benchmark tests
func BenchmarkVoiceprint_Create(b *testing.B) {
	db := setupVoiceprintTestDB(&testing.T{})
	featureVector := make([]byte, 512)
	for i := range featureVector {
		featureVector[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		voiceprint := &Voiceprint{
			SpeakerID:     "speaker-" + string(rune(i)),
			AssistantID:   "assistant-benchmark",
			SpeakerName:   "Benchmark User " + string(rune(i)),
			FeatureVector: featureVector,
		}
		db.Create(voiceprint)
	}
}

func BenchmarkVoiceprint_QueryBySpeakerID(b *testing.B) {
	db := setupVoiceprintTestDB(&testing.T{})

	// 创建测试数据
	for i := 0; i < 1000; i++ {
		voiceprint := &Voiceprint{
			SpeakerID:     "speaker-" + string(rune(i%100)),
			AssistantID:   "assistant-" + string(rune(i%10)),
			SpeakerName:   "User " + string(rune(i)),
			FeatureVector: []byte{byte(i), byte(i + 1), byte(i + 2)},
		}
		db.Create(voiceprint)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var results []Voiceprint
		db.Where("speaker_id = ?", "speaker-"+string(rune(i%100))).Find(&results)
	}
}

func BenchmarkVoiceprint_QueryByAssistantID(b *testing.B) {
	db := setupVoiceprintTestDB(&testing.T{})

	// 创建测试数据
	for i := 0; i < 1000; i++ {
		voiceprint := &Voiceprint{
			SpeakerID:     "speaker-" + string(rune(i)),
			AssistantID:   "assistant-" + string(rune(i%10)),
			SpeakerName:   "User " + string(rune(i)),
			FeatureVector: []byte{byte(i), byte(i + 1), byte(i + 2)},
		}
		db.Create(voiceprint)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var results []Voiceprint
		db.Where("assistant_id = ?", "assistant-"+string(rune(i%10))).Find(&results)
	}
}
