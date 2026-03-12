package voiceprint

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/cache"
	"go.uber.org/zap"
)

// Service 声纹识别服务
type Service struct {
	client *Client
	cache  cache.Cache
	config *Config
	logger *zap.Logger

	// 统计信息
	stats      *Statistics
	statsMutex sync.RWMutex
}

// NewService 创建新的声纹识别服务
func NewService(config *Config, cacheClient cache.Cache) (*Service, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create voiceprint client: %w", err)
	}

	service := &Service{
		client: client,
		cache:  cacheClient,
		config: config,
		logger: zap.L().Named("voiceprint.service"),
		stats: &Statistics{
			LastActivity: time.Now(),
		},
	}

	return service, nil
}

// IsEnabled 检查服务是否启用
func (s *Service) IsEnabled() bool {
	return s.client.IsEnabled()
}

// HealthCheck 健康检查
func (s *Service) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	if !s.IsEnabled() {
		return nil, ErrServiceDisabled
	}

	// 尝试从缓存获取
	if s.config.CacheEnabled {
		cacheKey := "voiceprint:health"
		if cached, found := s.cache.Get(ctx, cacheKey); found {
			if health, ok := cached.(*HealthResponse); ok {
				return health, nil
			}
		}
	}

	// 调用API
	health, err := s.client.HealthCheck(ctx)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	if s.config.CacheEnabled && health != nil {
		cacheKey := "voiceprint:health"
		s.cache.Set(ctx, cacheKey, health, 30*time.Second) // 健康检查缓存30秒
	}

	return health, nil
}

// RegisterVoiceprint 注册声纹
func (s *Service) RegisterVoiceprint(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	if !s.IsEnabled() {
		return nil, ErrServiceDisabled
	}

	startTime := time.Now()

	// 检查是否已存在
	if s.config.CacheEnabled {
		cacheKey := fmt.Sprintf("voiceprint:speaker:%s", req.SpeakerID)
		if exists := s.cache.Exists(ctx, cacheKey); exists {
			return nil, ErrSpeakerExists
		}
	}

	// 调用API注册
	resp, err := s.client.RegisterVoiceprint(ctx, req)
	if err != nil {
		s.updateStats(false, 0, time.Since(startTime))
		return nil, err
	}

	// 缓存注册信息
	if s.config.CacheEnabled && resp.Success {
		cacheKey := fmt.Sprintf("voiceprint:speaker:%s", req.SpeakerID)
		speakerInfo := &SpeakerInfo{
			SpeakerID:    req.SpeakerID,
			RegisterTime: time.Now(),
			UpdateTime:   time.Now(),
			Metadata:     req.Metadata,
		}
		s.cache.Set(ctx, cacheKey, speakerInfo, s.config.CacheTTL)
	}

	s.updateStats(resp.Success, 0, time.Since(startTime))

	if s.config.LogEnabled {
		s.logger.Info("Voiceprint registration completed",
			zap.String("speaker_id", req.SpeakerID),
			zap.Bool("success", resp.Success),
			zap.Duration("duration", time.Since(startTime)))
	}

	return resp, nil
}

// IdentifyVoiceprint 识别声纹
func (s *Service) IdentifyVoiceprint(ctx context.Context, req *IdentifyRequest) (*IdentifyResult, error) {
	if !s.IsEnabled() {
		return nil, ErrServiceDisabled
	}

	startTime := time.Now()

	// 设置默认阈值
	threshold := req.Threshold
	if threshold == 0 {
		threshold = s.config.SimilarityThreshold
	}

	// 调用API识别
	resp, err := s.client.IdentifyVoiceprint(ctx, req)
	if err != nil {
		s.updateStats(false, 0, time.Since(startTime))
		return nil, err
	}

	// 构建扩展结果
	result := &IdentifyResult{
		SpeakerID:   resp.SpeakerID,
		Score:       resp.Score,
		Confidence:  resp.Confidence,
		Threshold:   threshold,
		IsMatch:     resp.Score >= threshold,
		ProcessTime: time.Since(startTime),
		Timestamp:   time.Now(),
	}

	s.updateStats(true, resp.Score, time.Since(startTime))

	if s.config.LogEnabled {
		s.logger.Info("Voiceprint identification completed",
			zap.String("speaker_id", resp.SpeakerID),
			zap.Float64("score", resp.Score),
			zap.String("confidence", resp.Confidence),
			zap.Bool("is_match", result.IsMatch),
			zap.Float64("threshold", threshold),
			zap.Int("candidates", len(req.CandidateIDs)),
			zap.Duration("duration", time.Since(startTime)))
	}

	return result, nil
}

// IdentifyWithConfidence 带置信度检查的识别
func (s *Service) IdentifyWithConfidence(ctx context.Context, req *IdentifyRequest, minConfidence string) (*IdentifyResult, error) {
	result, err := s.IdentifyVoiceprint(ctx, req)
	if err != nil {
		return nil, err
	}

	// 检查置信度
	if !s.meetsConfidenceRequirement(result.Score, minConfidence) {
		return nil, ErrLowSimilarity
	}

	return result, nil
}

// DeleteVoiceprint 删除声纹
func (s *Service) DeleteVoiceprint(ctx context.Context, speakerID string) (*DeleteResponse, error) {
	if !s.IsEnabled() {
		return nil, ErrServiceDisabled
	}

	startTime := time.Now()

	// 调用API删除
	resp, err := s.client.DeleteVoiceprint(ctx, speakerID)
	if err != nil {
		return nil, err
	}

	// 清除缓存
	if s.config.CacheEnabled && resp.Success {
		cacheKey := fmt.Sprintf("voiceprint:speaker:%s", speakerID)
		s.cache.Delete(ctx, cacheKey)
	}

	if s.config.LogEnabled {
		s.logger.Info("Voiceprint deletion completed",
			zap.String("speaker_id", speakerID),
			zap.Bool("success", resp.Success),
			zap.Duration("duration", time.Since(startTime)))
	}

	return resp, nil
}

// BatchRegister 批量注册声纹
func (s *Service) BatchRegister(ctx context.Context, requests []RegisterRequest) (*BatchRegisterResponse, error) {
	if !s.IsEnabled() {
		return nil, ErrServiceDisabled
	}

	startTime := time.Now()
	results := make([]RegisterResponse, 0, len(requests))
	successCount := 0

	for _, req := range requests {
		resp, err := s.RegisterVoiceprint(ctx, &req)
		if err != nil {
			// 创建失败响应
			resp = &RegisterResponse{
				Success:   false,
				Message:   err.Error(),
				SpeakerID: req.SpeakerID,
				Timestamp: time.Now(),
			}
		} else if resp.Success {
			successCount++
		}
		results = append(results, *resp)
	}

	response := &BatchRegisterResponse{
		Success:   successCount,
		Failed:    len(requests) - successCount,
		Total:     len(requests),
		Results:   results,
		Timestamp: time.Now(),
	}

	if s.config.LogEnabled {
		s.logger.Info("Batch registration completed",
			zap.Int("total", len(requests)),
			zap.Int("success", successCount),
			zap.Int("failed", len(requests)-successCount),
			zap.Duration("duration", time.Since(startTime)))
	}

	return response, nil
}

// GetStatistics 获取统计信息
func (s *Service) GetStatistics() *Statistics {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()

	// 返回统计信息的副本
	stats := *s.stats
	return &stats
}

// updateStats 更新统计信息
func (s *Service) updateStats(success bool, score float64, duration time.Duration) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	s.stats.TotalIdentifications++
	s.stats.LastActivity = time.Now()

	if success {
		// 更新平均分数
		if s.stats.AverageScore == 0 {
			s.stats.AverageScore = score
		} else {
			s.stats.AverageScore = (s.stats.AverageScore + score) / 2
		}

		// 更新成功率
		successCount := float64(s.stats.TotalIdentifications) * s.stats.SuccessRate / 100
		successCount++
		s.stats.SuccessRate = (successCount / float64(s.stats.TotalIdentifications)) * 100
	} else {
		// 重新计算成功率
		if s.stats.TotalIdentifications > 1 {
			successCount := float64(s.stats.TotalIdentifications-1) * s.stats.SuccessRate / 100
			s.stats.SuccessRate = (successCount / float64(s.stats.TotalIdentifications)) * 100
		}
	}
}

// meetsConfidenceRequirement 检查是否满足置信度要求
func (s *Service) meetsConfidenceRequirement(score float64, minConfidence string) bool {
	switch minConfidence {
	case "very_high":
		return score >= 0.8
	case "high":
		return score >= 0.6
	case "medium":
		return score >= 0.4
	case "low":
		return score >= 0.2
	default:
		return score >= s.config.SimilarityThreshold
	}
}

// Close 关闭服务
func (s *Service) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
