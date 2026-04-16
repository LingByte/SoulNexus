package audio

import (
	"fmt"
)

const (
	// Provider kinds
	KindQiniu  = "qiniu"
	KindQCloud = "qcloud"
	KindAliyun = "aliyun"
)

// AudioCensor interface for audio content moderation
type AudioCensor interface {
	SubmitCensorAudio(audioURL string) (string, error)
	GetCensorResult(taskID string) (interface{}, error)
}

// GetAudioCensor returns an audio censor client based on the provider kind
func GetAudioCensor(kind string, credentials ...interface{}) (AudioCensor, error) {
	switch kind {
	case KindQiniu:
		if len(credentials) < 2 {
			return nil, fmt.Errorf("qiniu requires accessKey and secretKey")
		}
		accessKey, ok := credentials[0].(string)
		if !ok {
			return nil, fmt.Errorf("invalid accessKey type")
		}
		secretKey, ok := credentials[1].(string)
		if !ok {
			return nil, fmt.Errorf("invalid secretKey type")
		}
		return NewQiniuAudioCensor(accessKey, secretKey), nil

	case KindQCloud:
		if len(credentials) < 2 {
			return nil, fmt.Errorf("qcloud requires secretID and secretKey")
		}
		secretID, ok := credentials[0].(string)
		if !ok {
			return nil, fmt.Errorf("invalid secretID type")
		}
		secretKey, ok := credentials[1].(string)
		if !ok {
			return nil, fmt.Errorf("invalid secretKey type")
		}
		region := ""
		if len(credentials) > 2 {
			region, _ = credentials[2].(string)
		}
		return NewQCloudAudioCensor(secretID, secretKey, region)

	case KindAliyun:
		if len(credentials) < 2 {
			return nil, fmt.Errorf("aliyun requires accessKeyID and accessKeySecret")
		}
		accessKeyID, ok := credentials[0].(string)
		if !ok {
			return nil, fmt.Errorf("invalid accessKeyID type")
		}
		accessKeySecret, ok := credentials[1].(string)
		if !ok {
			return nil, fmt.Errorf("invalid accessKeySecret type")
		}
		endpoint := ""
		if len(credentials) > 2 {
			endpoint, _ = credentials[2].(string)
		}
		return NewAliyunAudioCensor(accessKeyID, accessKeySecret, endpoint)

	default:
		return nil, fmt.Errorf("unknown audio censor kind: %s", kind)
	}
}
