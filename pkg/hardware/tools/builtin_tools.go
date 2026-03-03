package tools

import (
	"fmt"
	"time"
)

// RegisterBuiltinTools 注册所有内置工具
func RegisterBuiltinTools(service *LLMService) {
	// 注册获取当前时间工具
	service.RegisterTool(
		"get_current_time",
		"获取当前时间，包括日期和时间",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"format": map[string]interface{}{
					"type":        "string",
					"description": "时间格式，可选值：datetime（日期+时间）、date（仅日期）、time（仅时间）",
					"enum":        []string{"datetime", "date", "time"},
				},
			},
		},
		executeGetCurrentTime,
	)

	// 注册获取天气工具（示例）
	service.RegisterTool(
		"get_weather",
		"获取指定城市的天气信息",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"city": map[string]interface{}{
					"type":        "string",
					"description": "城市名称，例如：北京、上海、深圳",
				},
			},
			"required": []string{"city"},
		},
		executeGetWeather,
	)
}

// executeGetCurrentTime 执行获取当前时间
func executeGetCurrentTime(args map[string]interface{}) (string, error) {
	format, _ := args["format"].(string)

	now := time.Now()
	var result string

	switch format {
	case "date":
		result = now.Format("2006-01-02")
	case "time":
		result = now.Format("15:04:05")
	case "datetime", "":
		result = now.Format("2006-01-02 15:04:05")
	default:
		result = now.Format("2006-01-02 15:04:05")
	}

	return result, nil
}

// executeGetWeather 执行获取天气（使用和风天气 API）
func executeGetWeather(args map[string]interface{}) (string, error) {
	city, ok := args["city"].(string)
	if !ok {
		return "", fmt.Errorf("缺少必需参数: city")
	}

	// 调用真实的天气 API
	result, err := GetWeather(city)
	if err != nil {
		// 如果 API 调用失败，返回友好的错误信息
		return fmt.Sprintf("抱歉，暂时无法获取%s的天气信息：%v", city, err), nil
	}

	return result, nil
}
