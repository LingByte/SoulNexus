package models

import (
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// AKSKRouteCatalogEntry describes one API-key-eligible HTTP capability.
type AKSKRouteCatalogEntry struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	GroupLabel  string `json:"groupLabel"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Permission  string `json:"permission,omitempty"`
}

func akskRoute(id, group, groupLabel, label, method, path, perm string) AKSKRouteCatalogEntry {
	return AKSKRouteCatalogEntry{
		ID: id, Group: group, GroupLabel: groupLabel, Label: label,
		Method: method, Path: path, Permission: perm,
	}
}

// AKSKRouteCatalog is the authoritative list of tenant-scoped endpoints that
// may be opened to API Key auth (platform-admin-only and human-JWT-only routes excluded).
var AKSKRouteCatalog = []AKSKRouteCatalogEntry{
	// Overview
	akskRoute("overview.stats", "overview", "概览", "仪表盘统计", "GET", "/overview/stats", ""),

	// AI reports
	akskRoute("reports.ai.list", "reports.ai", "AI 报告", "查询报告列表", "GET", "/reports/ai", "api.reports.read"),
	akskRoute("reports.ai.get", "reports.ai", "AI 报告", "查询报告详情", "GET", "/reports/ai/:id", "api.reports.read"),
	akskRoute("reports.ai.analytics", "reports.ai", "AI 报告", "会话分析图表", "GET", "/reports/ai/analytics", "api.reports.read"),
	akskRoute("reports.ai.caller_attributes", "reports.ai", "AI 报告", "主叫属性分析", "GET", "/reports/ai/analytics/caller-attributes", "api.reports.read"),
	akskRoute("reports.ai.caller_export", "reports.ai", "AI 报告", "导出主叫属性", "GET", "/reports/ai/analytics/caller-export", "api.reports.read"),
	akskRoute("reports.ai.overview", "reports.ai", "AI 报告", "报告概览", "GET", "/reports/ai/overview", "api.reports.read"),
	akskRoute("reports.ai.callin", "reports.ai", "AI 报告", "呼入分析", "GET", "/reports/ai/callin-analysis", "api.reports.read"),
	akskRoute("reports.ai.assistant", "reports.ai", "AI 报告", "智能体分析", "GET", "/reports/ai/assistant-analysis", "api.reports.read"),
	akskRoute("reports.ai.business", "reports.ai", "AI 报告", "业务分析", "GET", "/reports/ai/business-analysis", "api.reports.read"),
	akskRoute("reports.ai.key_findings", "reports.ai", "AI 报告", "关键发现", "GET", "/reports/ai/key-findings", "api.reports.read"),

	// Assistants / voices / voice clones / voiceprints
	akskRoute("assistants.list", "assistants", "智能体", "查询智能体列表", "GET", "/assistants", "api.assistants.read"),
	akskRoute("assistants.get", "assistants", "智能体", "查询智能体详情", "GET", "/assistants/:id", "api.assistants.read"),
	akskRoute("assistants.versions", "assistants", "智能体", "查询版本历史", "GET", "/assistants/:id/versions", "api.assistants.read"),
	akskRoute("assistants.diff", "assistants", "智能体", "对比版本差异", "GET", "/assistants/:id/diff", "api.assistants.read"),
	akskRoute("assistants.create", "assistants", "智能体", "创建智能体", "POST", "/assistants", "api.assistants.write"),
	akskRoute("assistants.update", "assistants", "智能体", "更新智能体", "PUT", "/assistants/:id", "api.assistants.write"),
	akskRoute("assistants.delete", "assistants", "智能体", "删除智能体", "DELETE", "/assistants/:id", "api.assistants.write"),
	akskRoute("assistants.publish", "assistants", "智能体", "发布智能体", "POST", "/assistants/:id/publish", "api.assistants.write"),
	akskRoute("assistants.rollback", "assistants", "智能体", "回滚智能体", "POST", "/assistants/:id/rollback", "api.assistants.write"),
	akskRoute("assistants.import", "assistants", "智能体", "从租户配置导入", "POST", "/assistants/import-from-tenant", "api.assistants.write"),
	akskRoute("assistants.members.list", "assistants", "智能体", "查询成员", "GET", "/assistants/:id/members", "api.assistants.read"),
	akskRoute("assistants.avatar", "assistants", "智能体", "上传头像", "POST", "/assistants/:id/avatar", "api.assistants.write"),
	akskRoute("assistants.members.update", "assistants", "智能体", "更新成员", "PUT", "/assistants/:id/members", "api.assistants.write"),
	akskRoute("assistants.members.add", "assistants", "智能体", "添加成员", "POST", "/assistants/:id/members", "api.assistants.write"),
	akskRoute("assistants.members.remove", "assistants", "智能体", "移除成员", "DELETE", "/assistants/:id/members/:userId", "api.assistants.write"),
	akskRoute("voices.list", "voices", "音色", "查询音色目录", "GET", "/voices", "api.assistants.read"),
	akskRoute("voices.preview", "voices", "音色", "试听音色", "POST", "/voices/preview", "api.assistants.read"),
	akskRoute("tenant_voice_providers.list", "voices", "音色", "查询租户语音供应商", "GET", "/tenant-voice-providers", "api.assistants.read"),
	akskRoute("voice_clones.config", "voice_clones", "音色克隆", "查询克隆配置", "GET", "/voice-clones/config", "api.assistants.read"),
	akskRoute("voice_clones.list", "voice_clones", "音色克隆", "列表", "GET", "/voice-clones", "api.assistants.read"),
	akskRoute("voice_clones.get", "voice_clones", "音色克隆", "查询详情", "GET", "/voice-clones/:id", "api.assistants.read"),
	akskRoute("voice_clones.training_texts", "voice_clones", "音色克隆", "训练文本", "GET", "/voice-clones/training-texts", "api.assistants.read"),
	akskRoute("voice_clones.create", "voice_clones", "音色克隆", "创建", "POST", "/voice-clones", "api.assistants.write"),
	akskRoute("voice_clones.audio", "voice_clones", "音色克隆", "上传训练音频", "POST", "/voice-clones/:id/audio", "api.assistants.write"),
	akskRoute("voice_clones.sync", "voice_clones", "音色克隆", "同步训练状态", "POST", "/voice-clones/:id/sync", "api.assistants.write"),
	akskRoute("voice_clones.preview", "voice_clones", "音色克隆", "试听", "POST", "/voice-clones/:id/preview", "api.assistants.read"),
	akskRoute("voice_clones.synthesize", "voice_clones", "音色克隆", "合成语音", "POST", "/voice-clones/:id/synthesize", "api.assistants.write"),
	akskRoute("voice_clones.delete", "voice_clones", "音色克隆", "删除", "DELETE", "/voice-clones/:id", "api.assistants.write"),
	akskRoute("voiceprints.config", "voiceprints", "声纹", "查询配置", "GET", "/voiceprints/config", "api.assistants.read"),
	akskRoute("voiceprints.self_test", "voiceprints", "声纹", "自检", "GET", "/voiceprints/self-test", "api.assistants.read"),
	akskRoute("voiceprints.list", "voiceprints", "声纹", "列表", "GET", "/voiceprints", "api.assistants.read"),
	akskRoute("voiceprints.get", "voiceprints", "声纹", "查询详情", "GET", "/voiceprints/:id", "api.assistants.read"),
	akskRoute("voiceprints.create", "voiceprints", "声纹", "创建", "POST", "/voiceprints", "api.assistants.write"),
	akskRoute("voiceprints.identify", "voiceprints", "声纹", "识别", "POST", "/voiceprints/identify", "api.assistants.write"),
	akskRoute("voiceprints.bind", "voiceprints", "声纹", "绑定智能体", "PUT", "/voiceprints/:id/assistant", "api.assistants.write"),
	akskRoute("voiceprints.delete", "voiceprints", "声纹", "删除", "DELETE", "/voiceprints/:id", "api.assistants.write"),
	akskRoute("voice_synthesis.list", "voice_synthesis", "语音合成", "历史列表", "GET", "/voice-synthesis-history", "api.assistants.read"),
	akskRoute("voice_synthesis.delete", "voice_synthesis", "语音合成", "删除历史", "DELETE", "/voice-synthesis-history/:id", "api.assistants.write"),

	// Knowledge base
	akskRoute("kb.list", "kb", "知识库", "查询知识库列表", "GET", "/knowledge-namespaces", "api.kb.read"),
	akskRoute("kb.get", "kb", "知识库", "查询知识库详情", "GET", "/knowledge-namespaces/:id", "api.kb.read"),
	akskRoute("kb.documents.list", "kb", "知识库", "查询文档列表", "GET", "/knowledge-namespaces/:id/documents", "api.kb.read"),
	akskRoute("kb.documents.get", "kb", "知识库", "查询文档详情", "GET", "/knowledge-namespaces/:id/documents/:docId", "api.kb.read"),
	akskRoute("kb.chunks.list", "kb", "知识库", "查询文档分块", "GET", "/knowledge-namespaces/:id/documents/:docId/chunks", "api.kb.read"),
	akskRoute("kb.chunks.get", "kb", "知识库", "查询单个分块", "GET", "/knowledge-namespaces/:id/documents/:docId/chunks/:chunkIndex", "api.kb.read"),
	akskRoute("kb.content", "kb", "知识库", "获取文档内容", "GET", "/knowledge-namespaces/:id/documents/:docId/content", "api.kb.read"),
	akskRoute("kb.recall", "kb", "知识库", "知识召回", "POST", "/knowledge-namespaces/:id/recall", "api.kb.read"),
	akskRoute("kb.create", "kb", "知识库", "创建知识库", "POST", "/knowledge-namespaces", "api.kb.write"),
	akskRoute("kb.update", "kb", "知识库", "更新知识库", "PUT", "/knowledge-namespaces/:id", "api.kb.write"),
	akskRoute("kb.delete", "kb", "知识库", "删除知识库", "DELETE", "/knowledge-namespaces/:id", "api.kb.write"),
	akskRoute("kb.documents.upload", "kb", "知识库", "上传文档", "POST", "/knowledge-namespaces/:id/documents", "api.kb.write"),
	akskRoute("kb.documents.update", "kb", "知识库", "更新文档", "PUT", "/knowledge-namespaces/:id/documents/:docId", "api.kb.write"),
	akskRoute("kb.documents.delete", "kb", "知识库", "删除文档", "DELETE", "/knowledge-namespaces/:id/documents/:docId", "api.kb.write"),
	akskRoute("kb.chunks.list_ns", "kb", "知识库", "查询命名空间分块", "GET", "/knowledge-namespaces/:id/chunks", "api.kb.read"),
	akskRoute("kb.chunks.export", "kb", "知识库", "导出分块", "GET", "/knowledge-namespaces/:id/chunks/export", "api.kb.read"),
	akskRoute("kb.chunks.create", "kb", "知识库", "创建分块", "POST", "/knowledge-namespaces/:id/chunks", "api.kb.write"),
	akskRoute("kb.chunks.update", "kb", "知识库", "更新分块", "PUT", "/knowledge-namespaces/:id/chunks/:chunkId", "api.kb.write"),
	akskRoute("kb.chunks.delete", "kb", "知识库", "删除分块", "DELETE", "/knowledge-namespaces/:id/chunks/:chunkId", "api.kb.write"),
	akskRoute("kb.unanswered.list", "kb", "知识库", "查询未答问题", "GET", "/knowledge-namespaces/:id/unanswered-questions", "api.kb.read"),
	akskRoute("kb.unanswered.count", "kb", "知识库", "未答问题计数", "GET", "/knowledge-namespaces/:id/unanswered-questions/count", "api.kb.read"),
	akskRoute("kb.unanswered.resolve", "kb", "知识库", "标记未答已解决", "POST", "/knowledge-namespaces/:id/unanswered-questions/:questionId/resolve", "api.kb.write"),
	akskRoute("kb.unanswered.delete", "kb", "知识库", "删除未答问题", "DELETE", "/knowledge-namespaces/:id/unanswered-questions/:questionId", "api.kb.write"),
	akskRoute("kb.hf.list", "kb", "知识库", "查询高频问题", "GET", "/knowledge-namespaces/:id/hf-questions", "api.kb.read"),
	akskRoute("kb.hf.daily_summary", "kb", "知识库", "高频问题日汇总", "GET", "/knowledge-namespaces/:id/hf-questions/daily-summary", "api.kb.read"),
	akskRoute("kb.hf.stats", "kb", "知识库", "高频问题统计", "GET", "/knowledge-namespaces/:id/hf-questions/:typicalId/stats", "api.kb.read"),
	akskRoute("kb.hf.answers", "kb", "知识库", "高频问题答案", "GET", "/knowledge-namespaces/:id/hf-questions/:typicalId/answers", "api.kb.read"),
	akskRoute("kb.analytics.quote_rate", "kb", "知识库", "引用率报告", "POST", "/knowledge-namespaces/:id/analytics/quote-rate", "api.kb.read"),
	akskRoute("kb.sync_sources.list", "kb", "知识库", "查询同步源", "GET", "/knowledge-namespaces/:id/sync-sources", "api.kb.read"),
	akskRoute("kb.sync_sources.create", "kb", "知识库", "创建同步源", "POST", "/knowledge-namespaces/:id/sync-sources", "api.kb.write"),
	akskRoute("kb.sync_sources.update", "kb", "知识库", "更新同步源", "PUT", "/knowledge-namespaces/:id/sync-sources/:sourceId", "api.kb.write"),
	akskRoute("kb.sync_sources.delete", "kb", "知识库", "删除同步源", "DELETE", "/knowledge-namespaces/:id/sync-sources/:sourceId", "api.kb.write"),
	akskRoute("kb.sync_sources.trigger", "kb", "知识库", "触发同步", "POST", "/knowledge-namespaces/:id/sync-sources/:sourceId/trigger", "api.kb.write"),
	akskRoute("kb.worker.stats", "kb", "知识库", "Worker 统计", "GET", "/knowledge-namespaces/:id/worker/stats", "api.kb.read"),
	akskRoute("kb.eval.datasets.list", "kb", "知识库", "查询评测数据集", "GET", "/knowledge-namespaces/:id/eval/datasets", "api.kb.read"),
	akskRoute("kb.eval.datasets.create", "kb", "知识库", "创建评测数据集", "POST", "/knowledge-namespaces/:id/eval/datasets", "api.kb.write"),
	akskRoute("kb.eval.datasets.delete", "kb", "知识库", "删除评测数据集", "DELETE", "/knowledge-namespaces/:id/eval/datasets/:datasetId", "api.kb.write"),
	akskRoute("kb.eval.run", "kb", "知识库", "运行评测", "POST", "/knowledge-namespaces/:id/eval/run", "api.kb.write"),
	akskRoute("kb.eval.compare", "kb", "知识库", "对比评测策略", "POST", "/knowledge-namespaces/:id/eval/compare", "api.kb.write"),
	akskRoute("kb.eval.jobs.get", "kb", "知识库", "查询评测任务", "GET", "/knowledge-namespaces/:id/eval/jobs/:jobId", "api.kb.read"),
	akskRoute("kb.documents.preview", "kb", "知识库", "文档预览", "GET", "/knowledge-namespaces/:id/documents/:docId/preview", "api.kb.read"),
	akskRoute("kb.documents.progress", "kb", "知识库", "索引进度", "GET", "/knowledge-namespaces/:id/documents/:docId/progress", "api.kb.read"),
	akskRoute("kb.documents.confirm_index", "kb", "知识库", "确认索引", "POST", "/knowledge-namespaces/:id/documents/:docId/confirm-index", "api.kb.write"),

	// JS templates
	akskRoute("js_templates.list", "js_templates", "JS 模板", "查询列表", "GET", "/js-templates", "api.assistants.read"),
	akskRoute("js_templates.get", "js_templates", "JS 模板", "查询详情", "GET", "/js-templates/:id", "api.assistants.read"),
	akskRoute("js_templates.create", "js_templates", "JS 模板", "创建", "POST", "/js-templates", "api.assistants.write"),
	akskRoute("js_templates.update", "js_templates", "JS 模板", "更新", "PUT", "/js-templates/:id", "api.assistants.write"),
	akskRoute("js_templates.delete", "js_templates", "JS 模板", "删除", "DELETE", "/js-templates/:id", "api.assistants.write"),

	// Workflows (read)
	akskRoute("workflow.definitions.list", "workflow", "工作流", "查询定义列表", "GET", "/workflows/definitions", "api.workflow.read"),
	akskRoute("workflow.definitions.get", "workflow", "工作流", "查询定义详情", "GET", "/workflows/definitions/:id", "api.workflow.read"),
	akskRoute("workflow.definitions.versions", "workflow", "工作流", "查询版本列表", "GET", "/workflows/definitions/:id/versions", "api.workflow.read"),
	akskRoute("workflow.definitions.version_get", "workflow", "工作流", "查询版本详情", "GET", "/workflows/definitions/:id/versions/:versionId", "api.workflow.read"),
	akskRoute("workflow.definitions.version_compare", "workflow", "工作流", "对比版本", "GET", "/workflows/definitions/:id/versions/compare", "api.workflow.read"),
	akskRoute("workflow.events.types", "workflow", "工作流", "事件类型", "GET", "/workflows/events/types", "api.workflow.read"),
	akskRoute("workflow.instances.list", "workflow", "工作流", "查询实例列表", "GET", "/workflows/instances", "api.workflow.read"),
	akskRoute("workflow.instances.export", "workflow", "工作流", "导出实例", "GET", "/workflows/instances/export", "api.workflow.read"),
	akskRoute("workflow.instances.get", "workflow", "工作流", "查询实例详情", "GET", "/workflows/instances/:id", "api.workflow.read"),

	// Workflow plugins (read)
	akskRoute("workflow_plugins.list", "workflow_plugins", "工作流插件", "查询列表", "GET", "/workflow-plugins", "api.workflow.read"),
	akskRoute("workflow_plugins.get", "workflow_plugins", "工作流插件", "查询详情", "GET", "/workflow-plugins/:id", "api.workflow.read"),
	akskRoute("workflow_plugins.published", "workflow_plugins", "工作流插件", "已发布插件", "GET", "/workflow-plugins/workflow/:workflowId/published", "api.workflow.read"),
	akskRoute("workflow_plugins.installed", "workflow_plugins", "工作流插件", "已安装插件", "GET", "/workflow-plugins/installed", "api.workflow.read"),
	akskRoute("workflow_plugins.my", "workflow_plugins", "工作流插件", "我的插件", "GET", "/workflow-plugins/my-plugins", "api.workflow.read"),

	// Node plugins (read)
	akskRoute("node_plugins.list", "node_plugins", "节点插件", "查询列表", "GET", "/node-plugins", "api.workflow.read"),
	akskRoute("node_plugins.get", "node_plugins", "节点插件", "查询详情", "GET", "/node-plugins/:id", "api.workflow.read"),
	akskRoute("node_plugins.installed", "node_plugins", "节点插件", "已安装插件", "GET", "/node-plugins/installed", "api.workflow.read"),

	// Billing
	akskRoute("billing.account", "billing", "账单", "查询账户信息", "GET", "/billing/account", "api.billing.read"),
	akskRoute("billing.usage.summary", "billing", "账单", "用量汇总", "GET", "/billing/usage/summary", "api.billing.read"),
	akskRoute("billing.usage.events", "billing", "账单", "用量明细", "GET", "/billing/usage/events", "api.billing.read"),
	akskRoute("billing.metrics", "billing", "账单", "业务指标", "GET", "/billing/metrics", "api.billing.read"),
	akskRoute("billing.bills.list", "billing", "账单", "查询账单列表", "GET", "/billing/bills", "api.billing.read"),
	akskRoute("billing.bills.get", "billing", "账单", "查询账单详情", "GET", "/billing/bills/:id", "api.billing.read"),
	akskRoute("billing.bills.export", "billing", "账单", "导出账单", "GET", "/billing/bills/:id/export", "api.billing.read"),
	akskRoute("billing.bills.finalize", "billing", "账单", "确认账单", "POST", "/billing/bills/:id/finalize", "api.billing.write"),
	akskRoute("billing.bills.mark_paid", "billing", "账单", "标记已付", "POST", "/billing/bills/:id/mark-paid", "api.billing.write"),

	// Tenant users
	akskRoute("tenant.users.list", "tenant.users", "成员", "查询成员列表", "GET", "/tenant-users", "api.tenant_users.read"),
	akskRoute("tenant.users.stats", "tenant.users", "成员", "成员统计", "GET", "/tenant-users/stats", "api.tenant_users.read"),
	akskRoute("tenant.users.get", "tenant.users", "成员", "查询成员详情", "GET", "/tenant-users/:id", "api.tenant_users.read"),
	akskRoute("tenant.users.create", "tenant.users", "成员", "创建成员", "POST", "/tenant-users", "api.tenant_users.write"),
	akskRoute("tenant.users.update", "tenant.users", "成员", "更新成员", "PUT", "/tenant-users/:id", "api.tenant_users.write"),
	akskRoute("tenant.users.status", "tenant.users", "成员", "更新成员状态", "PUT", "/tenant-users/:id/status", "api.tenant_users.write"),
	akskRoute("tenant.users.delete", "tenant.users", "成员", "删除成员", "DELETE", "/tenant-users/:id", "api.tenant_users.write"),
	akskRoute("tenant.users.restore", "tenant.users", "成员", "恢复成员", "POST", "/tenant-users/:id/restore", "api.tenant_users.write"),

	// Tenant org
	akskRoute("tenant.org.permissions", "tenant.org", "组织架构", "查询权限目录", "GET", "/tenant-org/permissions", "api.tenant_org.read"),
	akskRoute("tenant.org.groups.list", "tenant.org", "组织架构", "查询用户组", "GET", "/tenant-org/groups", "api.tenant_org.read"),
	akskRoute("tenant.org.roles.list", "tenant.org", "组织架构", "查询角色列表", "GET", "/tenant-org/roles", "api.tenant_org.read"),
	akskRoute("tenant.org.roles.get", "tenant.org", "组织架构", "查询角色详情", "GET", "/tenant-org/roles/:id", "api.tenant_org.read"),
	akskRoute("tenant.org.groups.create", "tenant.org", "组织架构", "创建用户组", "POST", "/tenant-org/groups", "api.tenant_org.write"),
	akskRoute("tenant.org.groups.update", "tenant.org", "组织架构", "更新用户组", "PUT", "/tenant-org/groups/:id", "api.tenant_org.write"),
	akskRoute("tenant.org.groups.delete", "tenant.org", "组织架构", "删除用户组", "DELETE", "/tenant-org/groups/:id", "api.tenant_org.write"),
	akskRoute("tenant.org.roles.create", "tenant.org", "组织架构", "创建角色", "POST", "/tenant-org/roles", "api.tenant_org.write"),
	akskRoute("tenant.org.roles.update", "tenant.org", "组织架构", "更新角色", "PUT", "/tenant-org/roles/:id", "api.tenant_org.write"),
	akskRoute("tenant.org.roles.delete", "tenant.org", "组织架构", "删除角色", "DELETE", "/tenant-org/roles/:id", "api.tenant_org.write"),
	akskRoute("tenant.org.roles.permissions", "tenant.org", "组织架构", "设置角色权限", "PUT", "/tenant-org/roles/:id/permissions", "api.tenant_org.write"),
	akskRoute("tenant.org.users.roles", "tenant.org", "组织架构", "设置用户角色", "PUT", "/tenant-org/users/:userId/roles", "api.tenant_org.write"),
	akskRoute("tenant.org.users.groups", "tenant.org", "组织架构", "设置用户分组", "PUT", "/tenant-org/users/:userId/groups", "api.tenant_org.write"),

	// Audit
	akskRoute("audit.logs.tenant", "audit", "操作日志", "查询租户操作日志", "GET", "/operation-logs/tenant", "api.operation_logs.read"),
	akskRoute("audit.ai_invocations.tenant", "audit", "AI 调用记录", "查询租户 AI 调用记录", "GET", "/ai-invocations/tenant", "api.ai_invocations.read"),

	// Voice session
	akskRoute("voice.session.create", "voice", "语音会话", "创建语音会话", "POST", "/lingecho/voice-session/v1/sessions", ""),
	akskRoute("voice.session.end", "voice", "语音会话", "结束语音会话", "DELETE", "/lingecho/voice-session/v1/sessions/:sessionId", ""),
	akskRoute("voice.session.webrtc", "voice", "语音会话", "WebRTC Offer", "POST", "/lingecho/voice-session/v1/webrtc/offer", ""),
	akskRoute("voice.session.ws", "voice", "语音会话", "WebSocket 连接", "GET", "/lingecho/voice-session/v1/ws", ""),
}

var akskCatalogByID map[string]AKSKRouteCatalogEntry

func init() {
	akskCatalogByID = make(map[string]AKSKRouteCatalogEntry, len(AKSKRouteCatalog))
	for _, e := range AKSKRouteCatalog {
		akskCatalogByID[e.ID] = e
	}
}

// AKSKCatalogEntryByID returns a catalog row by stable id.
func AKSKCatalogEntryByID(id string) (AKSKRouteCatalogEntry, bool) {
	e, ok := akskCatalogByID[strings.TrimSpace(id)]
	return e, ok
}

// AKSKCatalogGrouped returns catalog entries grouped for UI rendering.
func AKSKCatalogGrouped() []ginHGroup {
	order := []string{}
	seen := map[string]bool{}
	for _, e := range AKSKRouteCatalog {
		if !seen[e.Group] {
			seen[e.Group] = true
			order = append(order, e.Group)
		}
	}
	out := make([]ginHGroup, 0, len(order))
	for _, g := range order {
		grp := ginHGroup{ID: g, Label: "", Entries: nil}
		for _, e := range AKSKRouteCatalog {
			if e.Group != g {
				continue
			}
			if grp.Label == "" {
				grp.Label = e.GroupLabel
			}
			grp.Entries = append(grp.Entries, e)
		}
		out = append(out, grp)
	}
	return out
}

type ginHGroup struct {
	ID      string                  `json:"id"`
	Label   string                  `json:"label"`
	Entries []AKSKRouteCatalogEntry `json:"entries"`
}

// RoutePatternsForIDs resolves catalog ids to "METHOD /path" patterns.
func RoutePatternsForIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		e, ok := akskCatalogByID[id]
		if !ok {
			continue
		}
		out = append(out, e.Method+" "+e.Path)
	}
	return out
}

// NormalizeAKSKRouteIDs deduplicates and drops unknown ids.
func NormalizeAKSKRouteIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := akskCatalogByID[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// PermissionsForRouteIDs returns unique permission codes for selected routes.
func PermissionsForRouteIDs(ids []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, id := range ids {
		e, ok := akskCatalogByID[strings.TrimSpace(id)]
		if !ok || strings.TrimSpace(e.Permission) == "" {
			continue
		}
		if _, dup := seen[e.Permission]; dup {
			continue
		}
		seen[e.Permission] = struct{}{}
		out = append(out, e.Permission)
	}
	return out
}

// AKSKRoutePolicy controls which HTTP routes API Key credentials may access.
// JWT and platform-admin sessions are unaffected.
type AKSKRoutePolicy struct {
	Enabled  bool     `json:"enabled"`
	RouteIDs []string `json:"routeIds"`
	Routes   []string `json:"routes,omitempty"` // legacy raw patterns
}

var akskRoutePolicy atomic.Value // stores AKSKRoutePolicy

func defaultAKSKRoutePolicy() AKSKRoutePolicy {
	return AKSKRoutePolicy{Enabled: false, RouteIDs: nil}
}

// ReloadAKSKRoutePolicy refreshes the in-memory API Key route allowlist from
// system config (and env fallback). Call on startup and after config edits.
func ReloadAKSKRoutePolicy(db *gorm.DB) {
	policy := loadAKSKRoutePolicy(db)
	akskRoutePolicy.Store(policy)
}

// CurrentAKSKRoutePolicy returns the cached policy (deny-all when unset).
func CurrentAKSKRoutePolicy() AKSKRoutePolicy {
	if v := akskRoutePolicy.Load(); v != nil {
		if p, ok := v.(AKSKRoutePolicy); ok {
			return p
		}
	}
	return defaultAKSKRoutePolicy()
}

// SystemEnabledAKSKRouteIDs returns route ids open at platform level.
func SystemEnabledAKSKRouteIDs() []string {
	policy := CurrentAKSKRoutePolicy()
	if !policy.Enabled {
		return nil
	}
	if len(policy.RouteIDs) > 0 {
		return NormalizeAKSKRouteIDs(policy.RouteIDs)
	}
	return nil
}

func loadAKSKRoutePolicy(db *gorm.DB) AKSKRoutePolicy {
	raw := strings.TrimSpace(os.Getenv("API_AKSK_ROUTE_POLICY"))
	if db != nil {
		if v := strings.TrimSpace(utils.GetValue(db, constants.KEY_API_AKSK_ROUTE_POLICY)); v != "" {
			raw = v
		}
	}
	if raw == "" {
		return defaultAKSKRoutePolicy()
	}
	var policy AKSKRoutePolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return defaultAKSKRoutePolicy()
	}
	policy.RouteIDs = NormalizeAKSKRouteIDs(policy.RouteIDs)
	policy.Routes = normalizeAKSKRouteEntries(policy.Routes)
	return policy
}

func normalizeAKSKRouteEntries(routes []string) []string {
	if len(routes) == 0 {
		return nil
	}
	out := make([]string, 0, len(routes))
	for _, r := range routes {
		r = strings.TrimSpace(r)
		if r != "" {
			out = append(out, r)
		}
	}
	return out
}

func (p AKSKRoutePolicy) resolvedPatterns() []string {
	if len(p.RouteIDs) > 0 {
		return RoutePatternsForIDs(p.RouteIDs)
	}
	return p.Routes
}

// AKSKSystemRouteAllowed reports whether a request matches the platform allowlist.
func AKSKSystemRouteAllowed(method, requestPath string) bool {
	if os.Getenv("API_AKSK_ALLOW_ALL") == "true" {
		return true
	}
	policy := CurrentAKSKRoutePolicy()
	if !policy.Enabled {
		return false
	}
	patterns := policy.resolvedPatterns()
	if len(patterns) == 0 {
		return false
	}
	return akskPatternsAllow(method, requestPath, patterns)
}

// CredentialRoutesAllowed checks credential-scoped route ids against a request.
func CredentialRoutesAllowed(routeIDs []string, method, requestPath string) bool {
	routeIDs = NormalizeAKSKRouteIDs(routeIDs)
	if len(routeIDs) == 0 {
		return false
	}
	patterns := RoutePatternsForIDs(routeIDs)
	return akskPatternsAllow(method, requestPath, patterns)
}

// ValidateCredentialRouteIDs ensures every id is known and enabled at system level.
func ValidateCredentialRouteIDs(routeIDs []string) ([]string, error) {
	routeIDs = NormalizeAKSKRouteIDs(routeIDs)
	if len(routeIDs) == 0 {
		return nil, apperror.ErrAKSKRouteIDsRequired
	}
	enabled := SystemEnabledAKSKRouteIDs()
	if len(enabled) == 0 {
		return nil, apperror.ErrAKSKSystemRoutesClosed
	}
	allowed := map[string]struct{}{}
	for _, id := range enabled {
		allowed[id] = struct{}{}
	}
	for _, id := range routeIDs {
		if _, ok := allowed[id]; !ok {
			return nil, apperror.ErrAKSKRouteIDNotOpen
		}
	}
	return routeIDs, nil
}

func akskPatternsAllow(method, requestPath string, patterns []string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	rel := apiRelativePath(requestPath)
	full := normalizePath(requestPath)
	for _, entry := range patterns {
		if akskRouteEntryMatches(entry, method, rel, full) {
			return true
		}
	}
	return false
}

func apiRelativePath(requestPath string) string {
	prefix := ""
	if config.GlobalConfig != nil {
		prefix = strings.TrimSpace(config.GlobalConfig.Server.APIPrefix)
	}
	path := normalizePath(requestPath)
	if prefix == "" || prefix == "/" {
		return path
	}
	prefix = normalizePath(prefix)
	if path == prefix {
		return "/"
	}
	if strings.HasPrefix(path, prefix+"/") {
		return normalizePath(strings.TrimPrefix(path, prefix))
	}
	return path
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func akskRouteEntryMatches(entry, method, relPath, fullPath string) bool {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return false
	}
	parts := strings.Fields(entry)
	if len(parts) == 1 {
		return pathPatternMatches(parts[0], relPath) || pathPatternMatches(parts[0], fullPath)
	}
	entryMethod := strings.ToUpper(parts[0])
	pattern := strings.Join(parts[1:], " ")
	if entryMethod != method {
		return false
	}
	return pathPatternMatches(pattern, relPath) || pathPatternMatches(pattern, fullPath)
}

func pathPatternMatches(pattern, path string) bool {
	pattern = normalizePath(pattern)
	path = normalizePath(path)
	if pattern == "/*" || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	pp := splitPathSegments(pattern)
	rp := splitPathSegments(path)
	if len(pp) != len(rp) {
		return false
	}
	for i := range pp {
		if !segmentPatternMatches(pp[i], rp[i]) {
			return false
		}
	}
	return true
}

func splitPathSegments(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func segmentPatternMatches(pattern, segment string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, ":") {
		return segment != ""
	}
	return pattern == segment
}

// ParseAKSKRoutePolicyJSON validates and normalizes policy JSON from admin UI.
func ParseAKSKRoutePolicyJSON(raw string) (AKSKRoutePolicy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAKSKRoutePolicy(), nil
	}
	var policy AKSKRoutePolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return AKSKRoutePolicy{}, err
	}
	policy.RouteIDs = NormalizeAKSKRouteIDs(policy.RouteIDs)
	policy.Routes = normalizeAKSKRouteEntries(policy.Routes)
	return policy, nil
}

// MarshalAKSKRoutePolicy serializes policy for storage.
func MarshalAKSKRoutePolicy(policy AKSKRoutePolicy) (string, error) {
	policy.RouteIDs = NormalizeAKSKRouteIDs(policy.RouteIDs)
	policy.Routes = nil
	b, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func isAKSKRoutePolicyKey(key string) bool {
	return strings.EqualFold(strings.TrimSpace(key), constants.KEY_API_AKSK_ROUTE_POLICY)
}

// FilterCatalogGroupsByRouteIDs returns grouped catalog entries limited to ids.
func FilterCatalogGroupsByRouteIDs(ids []string) []ginHGroup {
	allow := map[string]struct{}{}
	for _, id := range NormalizeAKSKRouteIDs(ids) {
		allow[id] = struct{}{}
	}
	if len(allow) == 0 {
		return nil
	}
	all := AKSKCatalogGrouped()
	out := make([]ginHGroup, 0, len(all))
	for _, g := range all {
		grp := ginHGroup{ID: g.ID, Label: g.Label}
		for _, e := range g.Entries {
			if _, ok := allow[e.ID]; ok {
				grp.Entries = append(grp.Entries, e)
			}
		}
		if len(grp.Entries) > 0 {
			out = append(out, grp)
		}
	}
	return out
}
