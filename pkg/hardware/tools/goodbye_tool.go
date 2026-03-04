package tools

import (
	"fmt"
)

// GoodbyeCallback 待机回调函数类型
type GoodbyeCallback func() error

// RegisterGoodbyeTool 注册待机（goodbye）工具到 LLM
func RegisterGoodbyeTool(service *LLMService, callback GoodbyeCallback) {
	service.RegisterTool(
		"goodbye",
		"当用户表达任何形式的告别意图时，必须调用此函数。触发词包括但不限于：'再见'、'拜拜'、'bye'、'goodbye'、'我走了'、'我要走了'、'结束对话'、'结束'、'挂断'、'挂了'、'睡觉了'、'要睡了'、'晚安'、'886'、'88'、'跟我说再见'等。只要用户表达离开、结束、告别的意思，就应该调用此函数。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "告别的原因（可选）",
				},
			},
			"required": []string{},
		},
		func(args map[string]interface{}) (string, error) {
			return executeGoodbye(args, callback)
		},
	)

	service.logger.Info("已注册待机（goodbye）工具")
}

// executeGoodbye 执行待机操作
func executeGoodbye(args map[string]interface{}, callback GoodbyeCallback) (string, error) {
	reason, _ := args["reason"].(string)

	// 执行回调
	if callback != nil {
		if err := callback(); err != nil {
			return "", fmt.Errorf("设置待机状态失败: %w", err)
		}
	}

	// 返回成功消息（这个消息会被 TTS 播放）
	// 明确告诉 LLM：请将这段话说给用户听
	message := "好的，再见"
	if reason != "" {
		message = fmt.Sprintf("好的，%s，再见", reason)
	}

	// 返回格式：[SPEAK] 前缀提示 LLM 这是需要说出来的内容
	return fmt.Sprintf("[请对用户说] %s", message), nil
}
