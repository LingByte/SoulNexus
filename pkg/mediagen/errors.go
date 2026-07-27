package mediagen

import "errors"

var (
	ErrAPIKeyMissing   = errors.New("Seedream API Key 未配置，请设置环境变量 SEEDREAM_API_KEY")
	ErrInvalidImage    = errors.New("参考图不能为空或格式不支持")
	ErrEmptyPrompt     = errors.New("prompt 不能为空")
	ErrUpstreamFailed  = errors.New("上游生成服务请求失败")
	ErrNoImageData     = errors.New("未返回图片数据")
	ErrNoTaskID        = errors.New("未返回任务 ID")
)

type UpstreamError struct {
	Message string
}

func (e *UpstreamError) Error() string {
	if e == nil || e.Message == "" {
		return ErrUpstreamFailed.Error()
	}
	return e.Message
}
