package image

import (
	"fmt"
)

const (
	// Provider kinds
	KindQiniu  = "qiniu"
	KindQCloud = "qcloud"
	KindAliyun = "aliyun"
)

// ImageCensor interface for image content moderation
type ImageCensor interface {
	CensorImage(imageURL string) (interface{}, error)
}

// GetImageCensor returns an image censor client based on the provider kind
func GetImageCensor(kind string, credentials ...interface{}) (ImageCensor, error) {
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
		return NewQiniuImageCensor(accessKey, secretKey), nil

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
		return NewQCloudImageCensor(secretID, secretKey, region)

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
		return NewAliyunImageCensor(accessKeyID, accessKeySecret, endpoint)

	default:
		return nil, fmt.Errorf("unknown image censor kind: %s", kind)
	}
}
