package tools

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// SpeakerInfo 发音人信息
type SpeakerInfo struct {
	ID          string   `json:"id"`          // 发音人ID（如 101001）
	Name        string   `json:"name"`        // 发音人名称（如"智瑜"）
	Provider    string   `json:"provider"`    // TTS 提供商（qcloud, azure, baidu 等）
	Language    string   `json:"language"`    // 语言（zh-CN, en-US 等）
	Gender      string   `json:"gender"`      // 性别（male, female）
	Description string   `json:"description"` // 描述
	Keywords    []string `json:"keywords"`    // 关键词（用于匹配）
}

// SpeakerManager 发音人管理器
type SpeakerManager struct {
	speakers       map[string]*SpeakerInfo // ID -> SpeakerInfo
	currentID      string                  // 当前发音人ID
	switchCallback func(speakerID string) error
	mu             sync.RWMutex
	logger         *zap.Logger
}

// NewSpeakerManager 创建发音人管理器
func NewSpeakerManager(logger *zap.Logger) *SpeakerManager {
	mgr := &SpeakerManager{
		speakers: make(map[string]*SpeakerInfo),
		logger:   logger,
	}

	// 初始化默认发音人
	mgr.initDefaultSpeakers()

	return mgr
}

// initDefaultSpeakers 初始化默认发音人配置
func (m *SpeakerManager) initDefaultSpeakers() {
	defaultSpeakers := []*SpeakerInfo{
		// 腾讯云发音人
		{
			ID:          "101001",
			Name:        "智瑜（女声，温柔）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "温柔女声，适合日常对话",
			Keywords:    []string{"女声", "温柔", "标准", "普通话"},
		},
		{
			ID:          "101002",
			Name:        "智聆（女声，通用）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "通用女声，清晰自然",
			Keywords:    []string{"女声", "通用", "标准", "普通话"},
		},
		{
			ID:          "101003",
			Name:        "智美（女声，甜美）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "甜美女声，活泼可爱",
			Keywords:    []string{"女声", "甜美", "可爱", "普通话"},
		},
		{
			ID:          "101004",
			Name:        "智云（男声，成熟）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "male",
			Description: "成熟男声，沉稳大气",
			Keywords:    []string{"男声", "成熟", "沉稳", "普通话"},
		},
		{
			ID:          "101005",
			Name:        "智莉（女声，亲切）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "亲切女声，温暖友好",
			Keywords:    []string{"女声", "亲切", "温暖", "普通话"},
		},
		{
			ID:          "101006",
			Name:        "智言（男声，阳光）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "male",
			Description: "阳光男声，活力充沛",
			Keywords:    []string{"男声", "阳光", "活力", "普通话"},
		},
		{
			ID:          "101007",
			Name:        "智娜（女声，知性）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "知性女声，优雅大方",
			Keywords:    []string{"女声", "知性", "优雅", "普通话"},
		},
		{
			ID:          "101008",
			Name:        "智琪（女声，温暖）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "温暖女声，柔和舒适",
			Keywords:    []string{"女声", "温暖", "柔和", "普通话"},
		},
		{
			ID:          "101009",
			Name:        "智芸（女声，知性）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "知性女声，专业稳重",
			Keywords:    []string{"女声", "知性", "专业", "普通话"},
		},
		{
			ID:          "101010",
			Name:        "智华（男声，浑厚）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "male",
			Description: "浑厚男声，磁性有力",
			Keywords:    []string{"男声", "浑厚", "磁性", "普通话"},
		},
		// 四川话
		{
			ID:          "101040",
			Name:        "智瑜（四川话）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "female",
			Description: "四川话女声",
			Keywords:    []string{"四川话", "方言", "女声"},
		},
		// 粤语
		{
			ID:          "101019",
			Name:        "智瑜（粤语）",
			Provider:    "qcloud",
			Language:    "zh-HK",
			Gender:      "female",
			Description: "粤语女声",
			Keywords:    []string{"粤语", "广东话", "女声"},
		},
		// 英语
		{
			ID:          "101050",
			Name:        "WeJack（英语男声）",
			Provider:    "qcloud",
			Language:    "en-US",
			Gender:      "male",
			Description: "英语男声",
			Keywords:    []string{"英语", "english", "男声"},
		},
		// 童声
		{
			ID:          "101015",
			Name:        "智萌（童声）",
			Provider:    "qcloud",
			Language:    "zh-CN",
			Gender:      "child",
			Description: "童声，活泼可爱",
			Keywords:    []string{"童声", "儿童", "可爱"},
		},
	}

	for _, speaker := range defaultSpeakers {
		m.speakers[speaker.ID] = speaker
	}

	// 设置默认发音人
	m.currentID = "101001"
}

// SetSwitchCallback 设置切换回调
func (m *SpeakerManager) SetSwitchCallback(callback func(speakerID string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.switchCallback = callback
}

// GetCurrentSpeaker 获取当前发音人ID
func (m *SpeakerManager) GetCurrentSpeaker() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentID
}

// GetCurrentSpeakerInfo 获取当前发音人信息
func (m *SpeakerManager) GetCurrentSpeakerInfo() *SpeakerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.speakers[m.currentID]
}

// GetSpeaker 获取指定发音人信息
func (m *SpeakerManager) GetSpeaker(id string) *SpeakerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.speakers[id]
}

// ListSpeakers 列出所有发音人
func (m *SpeakerManager) ListSpeakers() []*SpeakerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	speakers := make([]*SpeakerInfo, 0, len(m.speakers))
	for _, speaker := range m.speakers {
		speakers = append(speakers, speaker)
	}
	return speakers
}

// FindSpeakerByKeyword 根据关键词查找发音人
func (m *SpeakerManager) FindSpeakerByKeyword(keyword string) *SpeakerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, speaker := range m.speakers {
		for _, kw := range speaker.Keywords {
			if kw == keyword {
				return speaker
			}
		}
	}
	return nil
}

// SwitchSpeaker 切换发音人
func (m *SpeakerManager) SwitchSpeaker(speakerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查发音人是否存在
	speaker, exists := m.speakers[speakerID]
	if !exists {
		return fmt.Errorf("发音人ID不存在: %s", speakerID)
	}

	// 检查是否需要切换
	if m.currentID == speakerID {
		return fmt.Errorf("当前已经是%s，无需切换", speaker.Name)
	}

	// 执行切换回调
	if m.switchCallback != nil {
		if err := m.switchCallback(speakerID); err != nil {
			return fmt.Errorf("切换失败: %w", err)
		}
	}

	// 更新当前发音人
	m.currentID = speakerID

	m.logger.Info("发音人切换成功",
		zap.String("speaker_id", speakerID),
		zap.String("speaker_name", speaker.Name))

	return nil
}

// RegisterSpeakerTool 注册发音人切换工具到 LLM
func RegisterSpeakerTool(service *LLMService, manager *SpeakerManager) {
	// 获取所有发音人信息，构建枚举
	speakers := manager.ListSpeakers()
	speakerOptions := make([]map[string]interface{}, 0, len(speakers))
	for _, speaker := range speakers {
		speakerOptions = append(speakerOptions, map[string]interface{}{
			"id":          speaker.ID,
			"name":        speaker.Name,
			"description": speaker.Description,
		})
	}

	// 构建发音人ID列表（用于枚举）
	speakerIDs := make([]string, 0, len(speakers))
	for _, speaker := range speakers {
		speakerIDs = append(speakerIDs, speaker.ID)
	}

	// 注册工具
	service.RegisterTool(
		"switch_speaker",
		"切换语音合成的发音人。当用户明确要求使用特定音色、性别、方言时调用。例如：'换个男声'、'用四川话说'、'换个温柔的声音'等。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"speaker_id": map[string]interface{}{
					"type":        "string",
					"description": fmt.Sprintf("发音人ID，可选值：%v", speakerOptions),
					"enum":        speakerIDs,
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "切换原因（可选）",
				},
			},
			"required": []string{"speaker_id"},
		},
		func(args map[string]interface{}) (string, error) {
			return executeSwitchSpeaker(args, manager)
		},
	)

	service.logger.Info("已注册发音人切换工具",
		zap.Int("speaker_count", len(speakers)))
}

// executeSwitchSpeaker 执行发音人切换
func executeSwitchSpeaker(args map[string]interface{}, manager *SpeakerManager) (string, error) {
	speakerID, ok := args["speaker_id"].(string)
	if !ok {
		return "", fmt.Errorf("缺少必需参数: speaker_id")
	}

	reason, _ := args["reason"].(string)

	// 获取发音人信息
	speaker := manager.GetSpeaker(speakerID)
	if speaker == nil {
		return "", fmt.Errorf("发音人ID不存在: %s", speakerID)
	}

	// 执行切换
	err := manager.SwitchSpeaker(speakerID)
	if err != nil {
		return fmt.Sprintf("切换失败：%v", err), nil
	}

	// 返回成功消息
	message := fmt.Sprintf("好的，已切换到%s", speaker.Name)
	if reason != "" {
		message += fmt.Sprintf("（%s）", reason)
	}

	return message, nil
}

// TTSSwitcher TTS 切换器接口
type TTSSwitcher interface {
	SwitchTTS(speakerID string) error
}
