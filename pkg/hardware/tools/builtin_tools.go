package tools

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

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

	// 注册搜索新闻工具
	service.RegisterTool(
		"search_news",
		"搜索最近的新闻资讯",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词，例如：科技、财经、体育等",
				},
			},
			"required": []string{"query"},
		},
		executeSearchNews,
	)

	// 注册搜索大众点评美食工具
	service.RegisterTool(
		"search_dianping_food",
		"搜索大众点评的城市美食推荐，包括店铺名称、地址、人均价格、评分等信息。使用前先说：您好，我这边帮您查一下大众点评，请稍后。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"city": map[string]interface{}{
					"type":        "string",
					"description": "城市名称，例如：北京、上海、成都、广州",
				},
			},
			"required": []string{"city"},
		},
		executeSearchDianpingFood,
	)
}

// executeGetCurrentTime 执行获取当前时间
func executeGetCurrentTime(args map[string]interface{}, llmService interface{}) (string, error) {
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
func executeGetWeather(args map[string]interface{}, llmService interface{}) (string, error) {
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

// executeSearchNews 执行搜索新闻
func executeSearchNews(args map[string]interface{}, llmService interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		query = "最新新闻" // 默认查询
	}

	// 返回提示信息，说明新闻查询功能暂未实现
	return fmt.Sprintf("抱歉，新闻查询功能暂时不可用。您可以访问新闻网站或使用其他新闻应用获取关于'%s'的最新资讯。", query), nil
}
