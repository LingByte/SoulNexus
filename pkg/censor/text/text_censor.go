package text

import (
	"fmt"
)

const (
	// Provider kinds
	KindAliyun = "aliyun"
	KindQCloud = "qcloud"
	KindQiNiu  = "qiniu"

	// Suggestion values
	SuggestionPass   = "pass"
	SuggestionReview = "review"
	SuggestionBlock  = "block"

	// Label values
	LabelNormal      = "normal"
	LabelSpam        = "spam"
	LabelAd          = "ad"
	LabelPolitics    = "politics"
	LabelTerrorism   = "terrorism"
	LabelAbuse       = "abuse"
	LabelPorn        = "porn"
	LabelFlood       = "flood"
	LabelContraband  = "contraband"
	LabelMeaningless = "meaningless"

	// Chinese messages for labels
	MsgNormal       = "正常文本"
	MsgSpam         = "含垃圾信息"
	MsgAd           = "广告"
	MsgPolitics     = "涉政"
	MsgTerrorism    = "暴恐"
	MsgAbuse        = "辱骂"
	MsgPorn         = "色情"
	MsgFlood        = "灌水"
	MsgContraband   = "违禁"
	MsgMeaningless  = "无意义"
	MsgUnknownLabel = "未知标签: %s"
)

// CensorResult represents the detailed text moderation result
type CensorResult struct {
	Suggestion string  `json:"suggestion"`        // pass, review, or block
	Label      string  `json:"label"`             // Moderation label: normal, spam, ad, politics, terrorism, abuse, porn, flood, contraband, etc.
	Score      float64 `json:"score"`             // Confidence score (0.0 to 1.0)
	Details    string  `json:"details,omitempty"` // Additional details or description
	Msg        string  `json:"msg"`               // Human-readable message based on label
}

// buildCensorMsg builds a human-readable message based on the label
func buildCensorMsg(label string) string {
	switch label {
	case LabelNormal:
		return MsgNormal
	case LabelSpam:
		return MsgSpam
	case LabelAd:
		return MsgAd
	case LabelPolitics:
		return MsgPolitics
	case LabelTerrorism:
		return MsgTerrorism
	case LabelAbuse:
		return MsgAbuse
	case LabelPorn:
		return MsgPorn
	case LabelFlood:
		return MsgFlood
	case LabelContraband:
		return MsgContraband
	case LabelMeaningless:
		return MsgMeaningless
	default:
		if label == "" {
			return MsgNormal
		}
		return fmt.Sprintf(MsgUnknownLabel, label)
	}
}

type TextCensor interface {
	CensorText(text string) (*CensorResult, error)
}

func GetTextCensor(kind string) (TextCensor, error) {
	if kind == "" {
		return NewQiniuTextCensor()
	}
	switch kind {
	case KindQiNiu:
		return NewQiniuTextCensor()
	case KindQCloud:
		return NewQCloudTextCensor()
	case KindAliyun:
		return NewAliyunTextCensor()
	}
	return nil, fmt.Errorf("unknown text censor kind: %s", kind)
}
