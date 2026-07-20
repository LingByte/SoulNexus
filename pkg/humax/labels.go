// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package humax

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Path segments that are actions, not resources (skip when building tags).
var actionSegments = map[string]bool{
	"revoke": true, "request": true, "cancel": true, "enable": true, "disable": true,
	"setup": true, "trust": true, "exchange": true, "callback": true, "login": true,
	"logout": true, "bind": true, "unbind": true, "send": true, "stats": true,
	"summary": true, "status": true, "public": true, "embed": true,
}

// areaZH maps first path area → Chinese sidebar group area.
var areaZH = map[string]string{
	"admin":                "平台管理",
	"platform":             "平台运维",
	"platform-admins":      "平台管理员",
	"system":               "系统",
	"me":                   "个人中心",
	"account":              "账号安全",
	"auth":                 "登录认证",
	"oauth":                "第三方登录",
	"tenants":              "租户",
	"tenant-users":         "租户成员",
	"tenant-org":           "组织架构",
	"credentials":          "访问凭证",
	"webhooks":             "Webhook",
	"operation-logs":       "操作日志",
	"knowledge-namespaces": "知识库",
	"ai-invocations":       "AI 调用",
	"ai":                   "AI",
	"billing":              "计费",
	"notifications":        "通知",
	"notification":         "通知",
	"workflows":            "工作流",
	"workflow-market":      "工作流市场",
	"workflow-plugins":     "工作流插件",
	"lingecho":             "呼叫中心",
	"voices":               "音色",
	"nlu-models":           "NLU 模型",
	"mcp-market":           "MCP 市场",
	"js-templates":         "JS 模板",
	"assistant-tools":      "助手工具",
	"node-plugins":         "节点插件",
	"reports":              "报表",
	"email-codes":          "邮箱验证码",
	"register":             "注册登录",
	"login":                "注册登录",
	"forgot-password":      "注册登录",
	"v1":                   "元信息",
	"meta":                 "元信息",
}

// resourceZH maps resource slug → short Chinese label (sidebar + summary).
var resourceZH = map[string]string{
	"execution-tasks":       "执行任务",
	"ai-invocations":        "AI 调用记录",
	"notification-channels": "通知渠道",
	"mail-templates":        "邮件模板",
	"mail-logs":             "邮件日志",
	"sms":                   "短信",
	"system-configs":        "系统配置",
	"platform-admins":       "平台管理员",
	"operation-logs":        "操作日志",
	"knowledge-namespaces":  "知识库命名空间",
	"tenant-users":          "租户成员",
	"tenant-org":            "组织架构",
	"credentials":           "访问凭证",
	"webhooks":              "Webhook",
	"devices":               "登录设备",
	"login-history":         "登录历史",
	"sessions":              "会话",
	"voiceprint":            "声纹",
	"totp":                  "二次验证 TOTP",
	"security-preferences":  "安全偏好",
	"avatar":                "头像",
	"password":              "密码",
	"email":                 "邮箱",
	"deletion":              "账号注销",
	"oauth":                 "OAuth",
	"github":                "GitHub",
	"tenants":               "租户",
	"billing":               "计费",
	"workflows":             "工作流",
	"workflow-market":       "工作流市场",
	"workflow-plugins":      "工作流插件",
	"js-templates":          "JS 模板",
	"assistant-tools":       "助手工具",
	"node-plugins":          "节点插件",
	"nlu-models":            "NLU 模型",
	"mcp-market":            "MCP 市场",
	"voices":                "音色",
	"reports":               "报表",
	"notifications":         "站内通知",
	"email-codes":           "邮箱验证码",
	"definitions":           "定义",
	"instances":             "实例",
	"category":              "分类",
	"sendcloud":             "SendCloud",
	"revoke":                "撤销",
	"me":                    "当前用户",
}

// actionVerbZH: last path segment → verb used in summary when not plain CRUD.
var actionVerbZH = map[string]string{
	"revoke":   "撤销",
	"request":  "申请",
	"cancel":   "取消",
	"enable":   "启用",
	"disable":  "关闭",
	"setup":    "配置",
	"trust":    "信任",
	"exchange": "兑换",
	"callback": "回调",
	"login":    "登录",
	"logout":   "退出登录",
	"bind":     "绑定",
	"unbind":   "解绑",
	"send":     "发送",
	"stats":    "统计",
	"summary":  "汇总",
	"status":   "状态",
}

// tagDescriptions explains each sidebar group (shown under Introduction / tag pages).
var tagDescriptions = map[string]string{
	"平台管理 · 执行任务":  "后台异步任务（导入、批处理、长耗时作业）的创建、查询与状态跟踪。对应控制台「系统 / 执行任务」。",
	"平台管理 · AI 调用记录": "平台侧查看各租户的大模型调用量、耗时与错误，用于成本与排障。",
	"平台管理 · 通知渠道":  "配置邮件 / 短信等通知通道（SMTP、厂商密钥等），供系统消息与告警使用。",
	"平台管理 · 邮件模板":  "系统邮件正文模板的增删改查。",
	"平台管理 · 邮件日志":  "已发送邮件的投递记录与统计。",
	"个人中心":         "当前登录用户的资料、安全设置、设备与会话。对应控制台「个人设置」。",
	"账号安全":         "账号注销申请 / 撤销等公开或半公开流程。",
	"登录认证":         "租户注册、登录、找回密码、退出。拿到 JWT 后才能调用其它需鉴权接口。",
	"访问凭证":         "API Access Key（AKSK）的创建与轮换，供程序化调用。",
	"Webhook":        "业务事件回调地址的订阅与管理。",
	"知识库":          "知识库命名空间、文档与检索相关接口。",
	"呼叫中心":         "语音对话与座席协助相关能力。",
	"租户":           "租户自身信息与租户级配置。",
	"租户成员":         "租户内用户的邀请、角色与禁用。",
	"操作日志":         "审计用操作日志查询。",
	"元信息":          "OpenAPI 文档元数据（非业务接口）。",
	"Meta":           "OpenAPI 文档元数据（非业务接口）。",
}

// InfoDescription is the OpenAPI Introduction shown at the top of /api/docs.
func InfoDescription(apiPrefix string) string {
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	return strings.TrimSpace(fmt.Sprintf(`
SoulNexus 后端 HTTP 接口手册。控制台、座席端、集成方都通过这些接口完成登录、呼叫、知识库、工作流等操作。

## 怎么用这份文档

1. **左侧按业务分组**（个人中心、登录认证、平台管理…），点开分组再选具体接口。
2. 点右上角 **Authorize**，填入登录拿到的 JWT：Authorization: Bearer <token>。
3. 在接口页点 **Try Request** 可直接试调（与控制台同一套鉴权）。
4. 需要离线集成时下载 [/openapi.json](/openapi.json) 或 [/openapi.yaml](/openapi.yaml)。

业务 API 前缀：%s。探活：/healthz · /livez · /readyz。

顶栏站点名与 Logo 来自公开接口 /system/init（与控制台相同的 SITE_NAME / SITE_LOGO_URL），不是进程配置里的服务名。
`, apiPrefix))
}

// HumanSummary builds a short Chinese sidebar title.
func HumanSummary(method, oapiPath string) string {
	segs := concreteSegs(oapiPath)
	if len(segs) == 0 {
		return method
	}
	last := segs[len(segs)-1]

	if verb, ok := actionVerbZH[last]; ok {
		base := resourceLabelZH(parentResource(segs))
		return compactSummary(verb + base)
	}

	resource := resourceLabelZH(last)
	singular := strings.TrimSuffix(resource, "列表")
	hasID := pathParamRe.MatchString(oapiPath)

	switch method {
	case http.MethodGet:
		if hasID {
			return compactSummary("获取" + singular)
		}
		return compactSummary("列出" + resource)
	case http.MethodPost:
		return compactSummary("创建" + singular)
	case http.MethodPut, http.MethodPatch:
		return compactSummary("更新" + singular)
	case http.MethodDelete:
		return compactSummary("删除" + singular)
	default:
		return compactSummary(method + " " + resource)
	}
}

func compactSummary(s string) string {
	// Keep sidebar titles short so Scalar does not ellipsize mid-word.
	r := []rune(s)
	if len(r) > 16 {
		return string(r[:15]) + "…"
	}
	return s
}

func humanDescription(method, oapiPath string) string {
	tag := HumanTag(oapiPath)
	sum := HumanSummary(method, oapiPath)
	desc := tagDescriptions[tag]
	if desc == "" {
		desc = "业务接口；请求/响应以实际 handler 与控制台行为为准。"
	}
	bodyHint := ""
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		bodyHint = "\n\n请求体若已注册类型，下方 Schema 会列出字段；否则需参考源码或控制台抓包。"
	}
	return fmt.Sprintf("**%s**\n\n路径：`%s %s`\n\n%s%s", sum, method, oapiPath, desc, bodyHint)
}

// HumanTag groups sidebar sections in Chinese, e.g. "平台管理 · 执行任务".
func HumanTag(oapiPath string) string {
	segs := concreteSegs(oapiPath)
	if len(segs) == 0 {
		return "其它"
	}

	// Drop trailing action verbs so /account/deletion/revoke → 账号安全 · 账号注销
	for len(segs) > 1 && actionSegments[segs[len(segs)-1]] {
		segs = segs[:len(segs)-1]
	}

	areaKey := segs[0]
	area := areaZH[areaKey]
	if area == "" {
		area = titleCaseWords(areaKey)
	}

	resKey := segs[len(segs)-1]
	// Prefer resource under admin/platform: admin/execution-tasks → 执行任务
	if len(segs) >= 2 && (areaKey == "admin" || areaKey == "platform" || areaKey == "system") {
		resKey = segs[len(segs)-1]
	} else if len(segs) == 1 {
		if d := tagDescriptions[area]; d != "" || areaZH[areaKey] != "" {
			return area
		}
		return resourceLabelZH(resKey)
	}

	res := resourceLabelZH(resKey)
	if res == area || resKey == areaKey {
		return area
	}
	return area + " · " + res
}

func TagDescription(name string) string {
	if d := tagDescriptions[name]; d != "" {
		return d
	}
	if i := strings.Index(name, " · "); i > 0 {
		area := name[:i]
		if d := tagDescriptions[area]; d != "" {
			return d
		}
		return area + "相关接口。左侧展开可查看具体操作。"
	}
	return name + "相关接口。左侧展开可查看具体操作。"
}

// EnsureTag registers tag metadata on the OpenAPI document (for Scalar intros).
func EnsureTag(api huma.API, name string) {
	if api == nil || name == "" {
		return
	}
	oapi := api.OpenAPI()
	for _, t := range oapi.Tags {
		if t != nil && t.Name == name {
			return
		}
	}
	oapi.Tags = append(oapi.Tags, &huma.Tag{
		Name:        name,
		Description: TagDescription(name),
	})
}

func concreteSegs(oapiPath string) []string {
	parts := strings.Split(strings.Trim(oapiPath, "/"), "/")
	var segs []string
	for _, p := range parts {
		if p == "" || p == "api" || strings.HasPrefix(p, "{") {
			continue
		}
		segs = append(segs, p)
	}
	return segs
}

func parentResource(segs []string) string {
	if len(segs) < 2 {
		if len(segs) == 1 {
			return segs[0]
		}
		return "资源"
	}
	// devices/{id}/revoke → devices
	for i := len(segs) - 2; i >= 0; i-- {
		if !actionSegments[segs[i]] {
			return segs[i]
		}
	}
	return segs[len(segs)-2]
}

func resourceLabelZH(slug string) string {
	if zh, ok := resourceZH[slug]; ok {
		return zh
	}
	return titleCaseWords(slug)
}
