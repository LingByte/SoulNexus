package notification

import "context"

type JPushConfig struct {
	AppKey       string
	MasterSecret string
}

type JPushClient interface {
	Push(ctx context.Context, title, content string, audience map[string]interface{}, extras map[string]interface{}) error
}

type JPush struct {
	cfg JPushConfig
	cli JPushClient
}

func NewJPush(cfg JPushConfig, cli JPushClient) *JPush { return &JPush{cfg: cfg, cli: cli} }

func (j *JPush) PushToAlias(ctx context.Context, alias []string, title, content string, extras map[string]interface{}) error {
	if j.cli == nil {
		return context.Canceled // 表示未配置客户端
	}
	aud := map[string]interface{}{"alias": alias}
	return j.cli.Push(ctx, title, content, aud, extras)
}

func (j *JPush) PushToAll(ctx context.Context, title, content string, extras map[string]interface{}) error {
	if j.cli == nil {
		return context.Canceled
	}
	aud := map[string]interface{}{"all": true}
	return j.cli.Push(ctx, title, content, aud, extras)
}
