package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVoiceTrainingTestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t,
		&User{},
		&VoiceTrainingTask{},
		&VoiceClone{},
		&VoiceSynthesis{},
		&VoiceTrainingText{},
		&VoiceTrainingTextSegment{},
	)
}

func TestVoiceTrainingTask_GetStatusText(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{
			name:   "in progress",
			status: TrainingStatusInProgress,
			want:   "训练中",
		},
		{
			name:   "failed",
			status: TrainingStatusFailed,
			want:   "失败",
		},
		{
			name:   "success",
			status: TrainingStatusSuccess,
			want:   "成功",
		},
		{
			name:   "queued",
			status: TrainingStatusQueued,
			want:   "排队中",
		},
		{
			name:   "unknown",
			status: 999,
			want:   "未知状态",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &VoiceTrainingTask{
				Status: tt.status,
			}
			assert.Equal(t, tt.want, task.GetStatusText())
		})
	}
}

func TestVoiceTrainingTask_GetSexText(t *testing.T) {
	tests := []struct {
		name string
		sex  int
		want string
	}{
		{
			name: "male",
			sex:  SexMale,
			want: "男性",
		},
		{
			name: "female",
			sex:  SexFemale,
			want: "女性",
		},
		{
			name: "unknown",
			sex:  999,
			want: "未知",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &VoiceTrainingTask{
				Sex: tt.sex,
			}
			assert.Equal(t, tt.want, task.GetSexText())
		})
	}
}

func TestVoiceTrainingTask_GetAgeGroupText(t *testing.T) {
	tests := []struct {
		name     string
		ageGroup int
		want     string
	}{
		{
			name:     "child",
			ageGroup: AgeGroupChild,
			want:     "儿童",
		},
		{
			name:     "youth",
			ageGroup: AgeGroupYouth,
			want:     "青年",
		},
		{
			name:     "middle",
			ageGroup: AgeGroupMiddle,
			want:     "中年",
		},
		{
			name:     "elderly",
			ageGroup: AgeGroupElderly,
			want:     "中老年",
		},
		{
			name:     "unknown",
			ageGroup: 999,
			want:     "未知",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &VoiceTrainingTask{
				AgeGroup: tt.ageGroup,
			}
			assert.Equal(t, tt.want, task.GetAgeGroupText())
		})
	}
}

func TestVoiceTrainingTask_IsCompleted(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{
			name:   "success",
			status: TrainingStatusSuccess,
			want:   true,
		},
		{
			name:   "failed",
			status: TrainingStatusFailed,
			want:   true,
		},
		{
			name:   "in progress",
			status: TrainingStatusInProgress,
			want:   false,
		},
		{
			name:   "queued",
			status: TrainingStatusQueued,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &VoiceTrainingTask{
				Status: tt.status,
			}
			assert.Equal(t, tt.want, task.IsCompleted())
		})
	}
}

func TestVoiceTrainingTask_IsSuccess(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{
			name:   "success",
			status: TrainingStatusSuccess,
			want:   true,
		},
		{
			name:   "failed",
			status: TrainingStatusFailed,
			want:   false,
		},
		{
			name:   "in progress",
			status: TrainingStatusInProgress,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &VoiceTrainingTask{
				Status: tt.status,
			}
			assert.Equal(t, tt.want, task.IsSuccess())
		})
	}
}

func TestVoiceClone_IncrementUsage(t *testing.T) {
	clone := &VoiceClone{
		UsageCount: 5,
	}

	clone.IncrementUsage()
	assert.Equal(t, 6, clone.UsageCount)
	assert.NotNil(t, clone.LastUsedAt)
}

func TestVoiceClone_IsAvailable(t *testing.T) {
	tests := []struct {
		name  string
		clone *VoiceClone
		want  bool
	}{
		{
			name: "available",
			clone: &VoiceClone{
				IsActive: true,
				AssetID:  "asset-123",
			},
			want: true,
		},
		{
			name: "not active",
			clone: &VoiceClone{
				IsActive: false,
				AssetID:  "asset-123",
			},
			want: false,
		},
		{
			name: "no asset ID",
			clone: &VoiceClone{
				IsActive: true,
				AssetID:  "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.clone.IsAvailable())
		})
	}
}

func TestVoiceTrainingTask_CRUD(t *testing.T) {
	db := setupVoiceTrainingTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	task := &VoiceTrainingTask{
		UserID:   user.ID,
		TaskID:   "task-123",
		TaskName: "Test Task",
		Status:   TrainingStatusInProgress,
		Sex:      SexMale,
		AgeGroup: AgeGroupYouth,
	}

	err = db.Create(task).Error
	require.NoError(t, err)
	assert.NotZero(t, task.ID)

	// Read
	var retrieved VoiceTrainingTask
	err = db.First(&retrieved, task.ID).Error
	require.NoError(t, err)
	assert.Equal(t, task.TaskID, retrieved.TaskID)

	// Update
	err = db.Model(&retrieved).Update("status", TrainingStatusSuccess).Error
	require.NoError(t, err)

	// Verify update
	err = db.First(&retrieved, task.ID).Error
	require.NoError(t, err)
	assert.Equal(t, TrainingStatusSuccess, retrieved.Status)
}

func TestVoiceClone_CRUD(t *testing.T) {
	db := setupVoiceTrainingTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	task := &VoiceTrainingTask{
		UserID:   user.ID,
		TaskID:   "task-123",
		TaskName: "Test Task",
		Status:   TrainingStatusSuccess,
	}
	err = db.Create(task).Error
	require.NoError(t, err)

	clone := &VoiceClone{
		UserID:         user.ID,
		TrainingTaskID: task.ID,
		AssetID:        "asset-123",
		TrainVID:       "vid-123",
		VoiceName:      "Test Voice",
		IsActive:       true,
	}

	err = db.Create(clone).Error
	require.NoError(t, err)
	assert.NotZero(t, clone.ID)

	// Read
	var retrieved VoiceClone
	err = db.First(&retrieved, clone.ID).Error
	require.NoError(t, err)
	assert.Equal(t, clone.AssetID, retrieved.AssetID)

	// Update usage
	retrieved.IncrementUsage()
	err = db.Save(&retrieved).Error
	require.NoError(t, err)

	// Verify update
	err = db.First(&retrieved, clone.ID).Error
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.UsageCount)
	assert.NotNil(t, retrieved.LastUsedAt)
}

func TestVoiceSynthesis_CRUD(t *testing.T) {
	db := setupVoiceTrainingTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	require.NoError(t, err)

	task := &VoiceTrainingTask{
		UserID:   user.ID,
		TaskID:   "task-123",
		TaskName: "Test Task",
		Status:   TrainingStatusSuccess,
	}
	err = db.Create(task).Error
	require.NoError(t, err)

	clone := &VoiceClone{
		UserID:         user.ID,
		TrainingTaskID: task.ID,
		AssetID:        "asset-123",
		VoiceName:      "Test Voice",
	}
	err = db.Create(clone).Error
	require.NoError(t, err)

	synthesis := &VoiceSynthesis{
		UserID:       user.ID,
		VoiceCloneID: clone.ID,
		Text:         "Hello world",
		Language:     "zh",
		AudioURL:     "http://example.com/audio.mp3",
		Status:       "success",
	}

	err = db.Create(synthesis).Error
	require.NoError(t, err)
	assert.NotZero(t, synthesis.ID)

	// Read
	var retrieved VoiceSynthesis
	err = db.First(&retrieved, synthesis.ID).Error
	require.NoError(t, err)
	assert.Equal(t, synthesis.Text, retrieved.Text)
}

func TestVoiceTrainingText_CRUD(t *testing.T) {
	db := setupVoiceTrainingTestDB(t)

	text := &VoiceTrainingText{
		TextID:   5001,
		TextName: "Test Text",
		Language: "zh",
		IsActive: true,
	}

	err := db.Create(text).Error
	require.NoError(t, err)
	assert.NotZero(t, text.ID)

	// Read
	var retrieved VoiceTrainingText
	err = db.First(&retrieved, text.ID).Error
	require.NoError(t, err)
	assert.Equal(t, text.TextID, retrieved.TextID)
}

func TestVoiceTrainingTextSegment_CRUD(t *testing.T) {
	db := setupVoiceTrainingTestDB(t)

	text := &VoiceTrainingText{
		TextID:   5001,
		TextName: "Test Text",
		Language: "zh",
	}
	err := db.Create(text).Error
	require.NoError(t, err)

	segment := &VoiceTrainingTextSegment{
		TextID:  text.ID,
		SegID:   "seg-1",
		SegText: "Segment text",
	}

	err = db.Create(segment).Error
	require.NoError(t, err)
	assert.NotZero(t, segment.ID)

	// Read
	var retrieved VoiceTrainingTextSegment
	err = db.First(&retrieved, segment.ID).Error
	require.NoError(t, err)
	assert.Equal(t, segment.SegID, retrieved.SegID)
	assert.Equal(t, segment.SegText, retrieved.SegText)
}
