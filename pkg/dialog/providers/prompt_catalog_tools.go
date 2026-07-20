package providers

import (
	"strings"
)

// CatalogToolsUsageHint returns a system-prompt appendix nudging the model to call tools.
func CatalogToolsUsageHint(toolNames []string) string {
	if len(toolNames) == 0 {
		return ""
	}
	// Highlight catalog-like tools (skip built-in transfer/knowledge names in prose).
	names := make([]string, 0, len(toolNames))
	for _, n := range toolNames {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		switch n {
		case "search_knowledge_base", "get_speaker_context", "identify_speaker":
			continue
		}
		names = append(names, n)
	}
	if len(names) == 0 {
		return ""
	}
	return "【工具调用规则】涉及时间/日期、学员、排课、预约、取消、课时、订单等业务时，" +
		"必须先调用已注册的 function tools（" + strings.Join(names, "、") + "），" +
		"禁止口头说「正在查询/已预约」而不发起 tool call；禁止凭记忆编造结果；拿到工具返回后再用自然语言回复用户。" +
		"语音提到学员名时：先 cloudsteps_resolve_student（或 list_students）用名下名单匹配 studentId，再 book_lesson；" +
		"已在名下不要 add_student；add_student 仅用于把新学员账号加到名下。" +
		"向用户播报学员时只念工具返回的 name（展示名），不要念 username/studentId。" +
		"涉及「今天/明天/几点」的预约：以系统提示【当前时间】为准推算日期，必要时先调 current_time；禁止臆造年份。"
}
