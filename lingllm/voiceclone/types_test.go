package voiceclone

import (
	"context"
	"testing"
	"time"
)

// MockVoiceCloneService is a mock implementation of VoiceCloneService for testing.
type MockVoiceCloneService struct {
	providerFunc         func() Provider
	getTrainingTextsFunc func(ctx context.Context, textID int64) (*TrainingText, error)
	createTaskFunc       func(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error)
	submitAudioFunc      func(ctx context.Context, req *SubmitAudioRequest) error
	queryTaskStatusFunc  func(ctx context.Context, taskID string) (*TaskStatus, error)
	synthesizeFunc       func(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error)
	synthesizeStreamFunc func(ctx context.Context, req *SynthesizeRequest, handler SynthesisHandler) error
}

func (m *MockVoiceCloneService) Provider() Provider {
	if m.providerFunc != nil {
		return m.providerFunc()
	}
	return ProviderXunfei
}

func (m *MockVoiceCloneService) GetTrainingTexts(ctx context.Context, textID int64) (*TrainingText, error) {
	if m.getTrainingTextsFunc != nil {
		return m.getTrainingTextsFunc(ctx, textID)
	}
	return nil, nil
}

func (m *MockVoiceCloneService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	if m.createTaskFunc != nil {
		return m.createTaskFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockVoiceCloneService) SubmitAudio(ctx context.Context, req *SubmitAudioRequest) error {
	if m.submitAudioFunc != nil {
		return m.submitAudioFunc(ctx, req)
	}
	return nil
}

func (m *MockVoiceCloneService) QueryTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	if m.queryTaskStatusFunc != nil {
		return m.queryTaskStatusFunc(ctx, taskID)
	}
	return nil, nil
}

func (m *MockVoiceCloneService) Synthesize(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockVoiceCloneService) SynthesizeStream(ctx context.Context, req *SynthesizeRequest, handler SynthesisHandler) error {
	if m.synthesizeStreamFunc != nil {
		return m.synthesizeStreamFunc(ctx, req, handler)
	}
	return nil
}

// MockSynthesisHandler is a mock implementation of SynthesisHandler.
type MockSynthesisHandler struct {
	onMessageFunc   func([]byte)
	onTimestampFunc func(SentenceTimestamp)
}

func (m *MockSynthesisHandler) OnMessage(data []byte) {
	if m.onMessageFunc != nil {
		m.onMessageFunc(data)
	}
}

func (m *MockSynthesisHandler) OnTimestamp(timestamp SentenceTimestamp) {
	if m.onTimestampFunc != nil {
		m.onTimestampFunc(timestamp)
	}
}

// Test Provider constants
func TestProviderConstants(t *testing.T) {
	if ProviderXunfei != "xunfei" {
		t.Errorf("ProviderXunfei = %s, want xunfei", ProviderXunfei)
	}
	if ProviderVolcengine != "volcengine" {
		t.Errorf("ProviderVolcengine = %s, want volcengine", ProviderVolcengine)
	}
}

// Test TrainingStatus constants
func TestTrainingStatusConstants(t *testing.T) {
	if TrainingStatusQueued != 2 {
		t.Errorf("TrainingStatusQueued = %d, want 2", TrainingStatusQueued)
	}
	if TrainingStatusInProgress != -1 {
		t.Errorf("TrainingStatusInProgress = %d, want -1", TrainingStatusInProgress)
	}
	if TrainingStatusSuccess != 1 {
		t.Errorf("TrainingStatusSuccess = %d, want 1", TrainingStatusSuccess)
	}
	if TrainingStatusFailed != 0 {
		t.Errorf("TrainingStatusFailed = %d, want 0", TrainingStatusFailed)
	}
}

// Test TrainingText struct
func TestTrainingText(t *testing.T) {
	segments := []TextSegment{
		{SegID: 1, SegText: "segment 1"},
		{SegID: "2", SegText: "segment 2"},
	}

	text := &TrainingText{
		TextID:   123,
		TextName: "test text",
		Segments: segments,
	}

	if text.TextID != 123 {
		t.Errorf("TextID = %d, want 123", text.TextID)
	}
	if text.TextName != "test text" {
		t.Errorf("TextName = %s, want test text", text.TextName)
	}
	if len(text.Segments) != 2 {
		t.Errorf("Segments length = %d, want 2", len(text.Segments))
	}
}

// Test CreateTaskRequest struct
func TestCreateTaskRequest(t *testing.T) {
	req := &CreateTaskRequest{
		TaskName:      "test task",
		Sex:           1,
		AgeGroup:      2,
		Language:      "zh",
		ResourceType:  12,
		EngineVersion: "omni_v1",
		Denoise:       1,
		MosRatio:      0.5,
	}

	if req.TaskName != "test task" {
		t.Errorf("TaskName = %s, want test task", req.TaskName)
	}
	if req.Sex != 1 {
		t.Errorf("Sex = %d, want 1", req.Sex)
	}
}

// Test SynthesizeRequest struct
func TestSynthesizeRequest(t *testing.T) {
	langID := 0
	req := &SynthesizeRequest{
		AssetID:    "asset123",
		Text:       "test text",
		Language:   "zh",
		LanguageID: &langID,
		Style:      "normal",
	}

	if req.AssetID != "asset123" {
		t.Errorf("AssetID = %s, want asset123", req.AssetID)
	}
	if req.Text != "test text" {
		t.Errorf("Text = %s, want test text", req.Text)
	}
	if req.LanguageID == nil || *req.LanguageID != 0 {
		t.Error("LanguageID should be 0")
	}
}

// Test SynthesizeResponse struct
func TestSynthesizeResponse(t *testing.T) {
	audioData := []byte{1, 2, 3, 4}
	resp := &SynthesizeResponse{
		AudioData:  audioData,
		Format:     "pcm",
		SampleRate: 16000,
		Duration:   1.5,
	}

	if len(resp.AudioData) != 4 {
		t.Errorf("AudioData length = %d, want 4", len(resp.AudioData))
	}
	if resp.Format != "pcm" {
		t.Errorf("Format = %s, want pcm", resp.Format)
	}
	if resp.SampleRate != 16000 {
		t.Errorf("SampleRate = %d, want 16000", resp.SampleRate)
	}
}

// Test TaskStatus struct
func TestTaskStatus(t *testing.T) {
	now := time.Now()
	status := &TaskStatus{
		TaskID:     "task123",
		TaskName:   "test task",
		Status:     TrainingStatusSuccess,
		AssetID:    "asset123",
		TrainVID:   "vid123",
		FailedDesc: "",
		Progress:   100.0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if status.TaskID != "task123" {
		t.Errorf("TaskID = %s, want task123", status.TaskID)
	}
	if status.Status != TrainingStatusSuccess {
		t.Errorf("Status = %d, want %d", status.Status, TrainingStatusSuccess)
	}
	if status.Progress != 100.0 {
		t.Errorf("Progress = %f, want 100.0", status.Progress)
	}
}

// Test SentenceTimestamp struct
func TestSentenceTimestamp(t *testing.T) {
	ts := SentenceTimestamp{
		StartTime: 1000,
		EndTime:   2000,
	}

	if ts.StartTime != 1000 {
		t.Errorf("StartTime = %d, want 1000", ts.StartTime)
	}
	if ts.EndTime != 2000 {
		t.Errorf("EndTime = %d, want 2000", ts.EndTime)
	}
}

// Test NewVoiceCloneSynthesisService
func TestNewVoiceCloneSynthesisService(t *testing.T) {
	mockService := &MockVoiceCloneService{
		providerFunc: func() Provider {
			return ProviderXunfei
		},
	}

	adapter := NewVoiceCloneSynthesisService(mockService, "asset123")

	if adapter.cloneService == nil {
		t.Error("cloneService should not be nil")
	}
	if adapter.assetID != "asset123" {
		t.Errorf("assetID = %s, want asset123", adapter.assetID)
	}
}

// Test VoiceCloneSynthesisService.Provider
func TestVoiceCloneSynthesisServiceProvider(t *testing.T) {
	mockService := &MockVoiceCloneService{
		providerFunc: func() Provider {
			return ProviderXunfei
		},
	}

	adapter := NewVoiceCloneSynthesisService(mockService, "asset123")
	provider := adapter.Provider()

	if provider != "xunfei" {
		t.Errorf("Provider = %s, want xunfei", provider)
	}
}

// Test VoiceCloneSynthesisService.CacheKey
func TestVoiceCloneSynthesisServiceCacheKey(t *testing.T) {
	mockService := &MockVoiceCloneService{}
	adapter := NewVoiceCloneSynthesisService(mockService, "asset123")

	key := adapter.CacheKey("test text")

	if key != "voiceclone_asset123_test text" {
		t.Errorf("CacheKey = %s, want voiceclone_asset123_test text", key)
	}
}

// Test VoiceCloneSynthesisService.Close
func TestVoiceCloneSynthesisServiceClose(t *testing.T) {
	mockService := &MockVoiceCloneService{}
	adapter := NewVoiceCloneSynthesisService(mockService, "asset123")

	err := adapter.Close()

	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

// Test MockSynthesisHandler
func TestMockSynthesisHandler(t *testing.T) {
	onMessageCalled := false
	onTimestampCalled := false

	handler := &MockSynthesisHandler{
		onMessageFunc: func(data []byte) {
			onMessageCalled = true
		},
		onTimestampFunc: func(ts SentenceTimestamp) {
			onTimestampCalled = true
		},
	}

	handler.OnMessage([]byte{1, 2, 3})
	handler.OnTimestamp(SentenceTimestamp{StartTime: 0, EndTime: 100})

	if !onMessageCalled {
		t.Error("OnMessage should have been called")
	}
	if !onTimestampCalled {
		t.Error("OnTimestamp should have been called")
	}
}

// Test Config struct
func TestConfig(t *testing.T) {
	config := &Config{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"app_id":  "test_app",
			"api_key": "test_key",
		},
	}

	if config.Provider != ProviderXunfei {
		t.Errorf("Provider = %s, want xunfei", config.Provider)
	}
	if len(config.Options) != 2 {
		t.Errorf("Options length = %d, want 2", len(config.Options))
	}
}
