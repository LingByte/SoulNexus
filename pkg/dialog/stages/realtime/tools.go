package realtime

import (
	"encoding/json"

	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"github.com/LingByte/lingllm/realtime"
)

var (
	voiceRealtimeGetCurrentTimeParams = json.RawMessage(`{
		"type":"object",
		"properties":{
			"timezone":{"type":"string","description":"IANA 时区，如 Asia/Shanghai；默认 Asia/Shanghai"}
		},
		"required":[],
		"additionalProperties":false
	}`)

	voiceRealtimeIsBusinessHoursParams = json.RawMessage(`{
		"type":"object",
		"properties":{
			"timezone":{"type":"string","description":"IANA 时区，默认 Asia/Shanghai"}
		},
		"required":[],
		"additionalProperties":false
	}`)

	voiceRealtimeCalculateParams = json.RawMessage(`{
		"type":"object",
		"properties":{
			"expression":{"type":"string","description":"简单算术表达式，仅支持数字与 + - * / 和括号，如 100+20*3"}
		},
		"required":["expression"],
		"additionalProperties":false
	}`)
)

// RealtimeTools returns tools registered on Qwen-Omni-Realtime session.update.
func RealtimeTools(includeKnowledge bool) []realtime.Tool {
	tools := []realtime.Tool{
		{
			Name:        "get_current_time",
			Description: "获取当前日期、时间与星期。用户问「几点了」「今天几号」「星期几」时调用。",
			Parameters:  voiceRealtimeGetCurrentTimeParams,
		},
		{
			Name:        "is_business_hours",
			Description: "判断当前是否在工作时间（周一至周五 9:00-18:00，指定时区）。用户问是否在营业时间时调用。",
			Parameters:  voiceRealtimeIsBusinessHoursParams,
		},
		{
			Name:        "calculate",
			Description: "计算简单算术表达式（仅 + - * / 与括号）。用户问心算类金额、数量时调用。",
			Parameters:  voiceRealtimeCalculateParams,
		},
	}
	if includeKnowledge {
		tools = append(tools, realtime.Tool{
			Name:        stageknow.SearchToolName,
			Description: stageknow.SearchToolDescription,
			Parameters:  stageknow.SearchToolParams,
		})
	}
	return tools
}

