package voiceprintconfig

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/lingllm/voiceprint"
)

// IdentifyOutcome is a normalized 1:N identification result.
type IdentifyOutcome struct {
	FeatureID  string  `json:"featureId"`
	Score      float64 `json:"score"`
	Threshold  float64 `json:"threshold"`
	IsMatch    bool    `json:"isMatch"`
	Confidence string  `json:"confidence,omitempty"`
}

// Bridge wraps provider-specific voiceprint operations for handlers.
type Bridge struct {
	provider voiceprint.Provider
	native   voiceprint.VoiceprintProvider
	httpSvc  *voiceprint.Service
	cfg      *voiceprint.Config
}

// NewBridge creates a configured voiceprint bridge.
func NewBridge() (*Bridge, error) {
	slug, provider, ok := ResolveEnabled()
	if !ok {
		return nil, fmt.Errorf("voiceprint not enabled or misconfigured")
	}
	b := &Bridge{provider: provider, cfg: HTTPServiceConfig()}
	switch provider {
	case voiceprint.ProviderHTTP:
		svc, err := voiceprint.NewService(b.cfg, noopCache{})
		if err != nil {
			return nil, err
		}
		b.httpSvc = svc
	default:
		factory := voiceprint.NewFactory()
		prov, err := factory.CreateProvider(&voiceprint.ProviderConfig{
			Provider: provider,
			Options:  optionsFromEnv(provider),
		})
		if err != nil {
			return nil, err
		}
		b.native = prov
	}
	_ = slug
	return b, nil
}

func (b *Bridge) Provider() voiceprint.Provider { return b.provider }

func (b *Bridge) SupportsIdentify() bool {
	return b.provider == voiceprint.ProviderHTTP ||
		b.provider == voiceprint.ProviderXunfei ||
		b.provider == voiceprint.ProviderVolcengine
}

func (b *Bridge) SimilarityThreshold() float64 {
	if b.cfg != nil && b.cfg.SimilarityThreshold > 0 {
		return b.cfg.SimilarityThreshold
	}
	return 0.6
}

func (b *Bridge) EnsureTenantGroup(ctx context.Context, tenantID uint, name string) error {
	if b.native == nil {
		return nil
	}
	if b.provider == voiceprint.ProviderVolcengine {
		return nil
	}
	groupID := TenantGroupID(tenantID)
	_, err := b.native.CreateGroup(ctx, groupID, name, fmt.Sprintf("tenant=%d", tenantID))
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "exist") || strings.Contains(msg, "already") || strings.Contains(msg, "重复") {
		return nil
	}
	return err
}

func (b *Bridge) Enroll(
	ctx context.Context,
	tenantID uint,
	assistantID *uint,
	profileID uint,
	featureID, displayName, description string,
	audio []byte,
) (string, error) {
	if err := voiceprint.ValidateWAVFormat(audio); err != nil {
		return "", err
	}
	switch b.provider {
	case voiceprint.ProviderHTTP:
		if b.httpSvc == nil {
			return "", voiceprint.ErrServiceDisabled
		}
		resp, err := b.httpSvc.RegisterVoiceprint(ctx, &voiceprint.RegisterRequest{
			ProfileID:   ProfileIDString(profileID),
			FeatureID:   featureID,
			SpeakerID:   featureID,
			TenantID:    TenantIDString(tenantID),
			AssistantID: AssistantIDString(assistantID),
			Name:        displayName,
			Provider:    string(b.provider),
			Status:      "active",
			Description: description,
			AudioData:   audio,
		})
		if err != nil {
			return "", err
		}
		if !resp.Success {
			msg := strings.TrimSpace(resp.Message)
			if msg == "" {
				msg = "voiceprint microservice rejected enrollment"
			}
			return "", fmt.Errorf("%s", msg)
		}
		return featureID, nil
	default:
		if err := b.EnsureTenantGroup(ctx, tenantID, displayName); err != nil {
			return "", err
		}
		out, err := b.native.CreateFeature(ctx, TenantGroupID(tenantID), featureID, displayName, audio)
		if err != nil {
			return "", err
		}
		if out != nil && strings.TrimSpace(out.FeatureID) != "" {
			return strings.TrimSpace(out.FeatureID), nil
		}
		return featureID, nil
	}
}

func (b *Bridge) BindAssistant(ctx context.Context, tenantID uint, featureID string, assistantID *uint) error {
	switch b.provider {
	case voiceprint.ProviderHTTP:
		// 元数据由 Go 写入同一张 voiceprints 表，无需再调 HTTP。
		return nil
	default:
		return nil
	}
}

func (b *Bridge) Identify(
	ctx context.Context,
	tenantID uint,
	assistantID *uint,
	candidates []string,
	audio []byte,
	threshold float64,
) (*IdentifyOutcome, error) {
	if !b.SupportsIdentify() {
		return nil, fmt.Errorf("provider %s does not support 1:N identify", b.provider)
	}
	if err := voiceprint.ValidateWAVFormat(audio); err != nil {
		return nil, err
	}
	if threshold <= 0 {
		threshold = b.SimilarityThreshold()
	}
	switch b.provider {
	case voiceprint.ProviderHTTP:
		if b.httpSvc == nil {
			return nil, voiceprint.ErrServiceDisabled
		}
		result, err := b.httpSvc.IdentifyVoiceprint(ctx, &voiceprint.IdentifyRequest{
			CandidateIDs: candidates,
			TenantID:     TenantIDString(tenantID),
			AssistantID:  AssistantIDString(assistantID),
			AudioData:    audio,
			Threshold:    threshold,
		})
		if err != nil {
			return nil, err
		}
		return &IdentifyOutcome{
			FeatureID:  result.SpeakerID,
			Score:      result.Score,
			Threshold:  threshold,
			IsMatch:    result.IsMatch,
			Confidence: result.Confidence,
		}, nil
	default:
		topK := len(candidates)
		if topK <= 0 {
			topK = 5
		}
		if b.provider == voiceprint.ProviderVolcengine {
			if ve, ok := b.native.(*voiceprint.VolcengineProviderAdapter); ok {
				out, err := ve.IdentifyAudio(ctx, candidates, topK, audio)
				if err != nil {
					return nil, err
				}
				if out == nil || len(out.ScoreList) == 0 {
					return &IdentifyOutcome{Threshold: threshold, IsMatch: false}, nil
				}
				best := out.ScoreList[0]
				return &IdentifyOutcome{
					FeatureID: best.FeatureID,
					Score:     best.Score,
					Threshold: threshold,
					IsMatch:   best.Score >= threshold,
				}, nil
			}
		}
		out, err := b.native.SearchFea(ctx, TenantGroupID(tenantID), topK, audio)
		if err != nil {
			return nil, err
		}
		if out == nil || len(out.ScoreList) == 0 {
			return &IdentifyOutcome{Threshold: threshold, IsMatch: false}, nil
		}
		best := out.ScoreList[0]
		score := best.Score
		return &IdentifyOutcome{
			FeatureID: best.FeatureID,
			Score:     score,
			Threshold: threshold,
			IsMatch:   score >= threshold,
		}, nil
	}
}

func (b *Bridge) Delete(ctx context.Context, tenantID uint, assistantID *uint, featureID string) error {
	switch b.provider {
	case voiceprint.ProviderHTTP:
		// 行由 Go 从 voiceprints 表删除（含 feature_vector）。
		return nil
	default:
		return b.native.DeleteFeature(ctx, TenantGroupID(tenantID), featureID)
	}
}

func (b *Bridge) HealthCheck(ctx context.Context) error {
	switch b.provider {
	case voiceprint.ProviderHTTP:
		if b.httpSvc == nil {
			return voiceprint.ErrServiceDisabled
		}
		_, err := b.httpSvc.HealthCheck(ctx)
		return err
	default:
		return b.native.HealthCheck(ctx)
	}
}

func (b *Bridge) Close() error {
	if b.native != nil {
		return b.native.Close()
	}
	if b.httpSvc != nil {
		return b.httpSvc.Close()
	}
	return nil
}

type noopCache struct{}

func (noopCache) Get(context.Context, string) (interface{}, bool) { return nil, false }
func (noopCache) Set(context.Context, string, interface{}, time.Duration) error { return nil }
func (noopCache) Exists(context.Context, string) bool                           { return false }
func (noopCache) Delete(context.Context, string) error                          { return nil }
