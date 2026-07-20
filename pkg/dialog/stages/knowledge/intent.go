package knowledge

import "strings"

var searchIntentKeywords = []string{
	"检索", "查询", "查一下", "查下", "搜索", "搜一下", "知识库",
	"介绍", "了解一下", "了解", "什么是", "啥是", "是什么",
	"怎么", "如何", "怎样", "多少钱", "价格", "费用", "优惠",
	"办理", "流程", "政策", "规则", "工单", "账号", "密码",
	"大赛", "项目", "产品", "功能", "服务", "备案", "域名", "证书",
	"赛道", "获奖", "奖项", "人员", "员工", "实习",
	"403", "429", "存储", "cdn", "dcdn", "sdk", "wss",
}

var searchSkipPhrases = []string{
	"听得到", "听得清", "能听到", "可以听到", "说说话", "说话吗",
	"在吗", "在不在", "有人吗",
	"你好", "您好", "早上好", "晚上好",
	"再见", "拜拜",
	"谢谢", "感谢",
	"知道了", "明白了", "没问题",
	"联系客服", "人工协助",
}

// ShouldRunSearch reports whether optional server-side pre-enrich heuristics apply.
func ShouldRunSearch(query string) bool {
	q := normalizeQuery(query)
	if q == "" {
		return false
	}
	runes := []rune(q)
	if len(runes) <= 1 {
		return false
	}
	hasIntent := queryHasIntent(q)
	if queryIsChitchat(q) && !hasIntent {
		return false
	}
	if len(runes) <= 4 && !hasIntent {
		return false
	}
	return true
}

func normalizeQuery(query string) string {
	q := strings.TrimSpace(query)
	q = strings.TrimRight(q, "。！？!?.,， ")
	return q
}

func queryHasIntent(q string) bool {
	for _, k := range searchIntentKeywords {
		if strings.Contains(q, k) {
			return true
		}
	}
	return false
}

func queryIsChitchat(q string) bool {
	for _, p := range searchSkipPhrases {
		if strings.Contains(q, p) {
			return true
		}
	}
	switch q {
	case "好的", "好", "嗯", "嗯嗯", "哦", "啊", "呃", "这个", "那个", "是的", "对":
		return true
	}
	if strings.Contains(q, "听到") && (strings.Contains(q, "吗") || strings.Contains(q, "？") || strings.Contains(q, "?")) {
		return true
	}
	return false
}

var queryOralFillers = []string{
	"嗯啊", "嗯", "啊", "呃", "哦", "唉",
	"你那个什么", "我的那个", "就是那个", "那个什么",
	"我想问一下", "我想问", "我想", "帮我", "请帮我", "请问",
	"查一下", "查询一下", "检索一下", "搜索一下",
	"对，", "对,", "嗯，", "嗯,", "啊，", "啊,",
}

// UserUtteranceForSearch returns the primary user utterance for KB recall.
// Strips LLM-only system blocks (NLU hints, prior KB inject) that must not be embedded/searched.
func UserUtteranceForSearch(raw string) string {
	q := strings.TrimSpace(raw)
	if q == "" {
		return ""
	}
	for _, sep := range []string{
		"\n\n【系统·NLU】",
		"\n\n【系统·",
		"\n\n[系统知识库检索",
		"\n\n【知识库",
	} {
		if i := strings.Index(q, sep); i >= 0 {
			q = strings.TrimSpace(q[:i])
		}
	}
	return q
}

// CompactSearchQuery strips oral fillers from ASR text for sharper recall.
func CompactSearchQuery(raw string) string {
	q := normalizeQuery(UserUtteranceForSearch(raw))
	if q == "" {
		return ""
	}
	for _, f := range queryOralFillers {
		q = strings.ReplaceAll(q, f, "")
	}
	q = strings.TrimSpace(q)
	q = strings.TrimRight(q, "，,、 ")
	if runes := []rune(q); len(runes) > 96 {
		q = string(runes[len(runes)-96:])
	}
	return strings.TrimSpace(q)
}

// ShouldServerEnrich reports whether pipeline mode should pre-inject KB context.
// Realtime omni does not use this — the model calls search_knowledge_base when needed.
func ShouldServerEnrich(callID, query string) bool {
	if !ResolveBinding(callID).Enabled {
		return false
	}
	if !ShouldRunSearch(query) {
		return false
	}
	cfg := ResolveSearchConfig(callID)
	if cfg.AutoEnrich {
		return true
	}
	// AutoEnrich off: still enrich when the user explicitly asks for KB/search.
	q := normalizeQuery(query)
	return strings.Contains(q, "知识库") || strings.Contains(q, "检索") || strings.Contains(q, "查询")
}
