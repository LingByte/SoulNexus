package sip

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/emiago/sipgo/sip"
	"github.com/sirupsen/logrus"
)

// RoutingRule 路由规则
type RoutingRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Priority    int      `json:"priority"`    // 优先级，数字越小优先级越高
	Pattern     string   `json:"pattern"`     // 匹配模式（正则表达式）
	Target      string   `json:"target"`      // 目标URI或路由目标
	Action      string   `json:"action"`      // route, reject, redirect
	Enabled     bool     `json:"enabled"`     // 是否启用
	Description string   `json:"description"` // 描述
	Conditions  []string `json:"conditions"`  // 额外条件
}

// RoutingTable 路由表
type RoutingTable struct {
	rules []*RoutingRule
	mutex sync.RWMutex
}

// NewRoutingTable 创建路由表
func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		rules: make([]*RoutingRule, 0),
	}
}

// AddRule 添加路由规则
func (rt *RoutingTable) AddRule(rule *RoutingRule) error {
	// 验证正则表达式
	if _, err := regexp.Compile(rule.Pattern); err != nil {
		return fmt.Errorf("invalid pattern: %v", err)
	}

	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	rt.rules = append(rt.rules, rule)
	// 按优先级排序
	rt.sortRules()

	logrus.WithFields(logrus.Fields{
		"rule_id": rule.ID,
		"name":    rule.Name,
		"pattern": rule.Pattern,
	}).Info("Routing rule added")

	return nil
}

// RemoveRule 移除路由规则
func (rt *RoutingTable) RemoveRule(ruleID string) error {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	for i, rule := range rt.rules {
		if rule.ID == ruleID {
			rt.rules = append(rt.rules[:i], rt.rules[i+1:]...)
			logrus.WithField("rule_id", ruleID).Info("Routing rule removed")
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", ruleID)
}

// FindRoute 查找匹配的路由规则
func (rt *RoutingTable) FindRoute(fromURI, toURI string) (*RoutingRule, error) {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	// 按优先级顺序查找
	for _, rule := range rt.rules {
		if !rule.Enabled {
			continue
		}

		// 匹配模式
		matched, err := rt.matchPattern(rule.Pattern, fromURI, toURI)
		if err != nil {
			logrus.WithError(err).WithField("rule_id", rule.ID).Error("Pattern match error")
			continue
		}

		if matched {
			// 检查额外条件
			if rt.checkConditions(rule.Conditions, fromURI, toURI) {
				logrus.WithFields(logrus.Fields{
					"rule_id": rule.ID,
					"from":    fromURI,
					"to":      toURI,
				}).Info("Route matched")
				return rule, nil
			}
		}
	}

	return nil, fmt.Errorf("no matching route found")
}

// matchPattern 匹配模式
func (rt *RoutingTable) matchPattern(pattern, fromURI, toURI string) (bool, error) {
	// 支持变量替换
	pattern = strings.ReplaceAll(pattern, "${from}", regexp.QuoteMeta(fromURI))
	pattern = strings.ReplaceAll(pattern, "${to}", regexp.QuoteMeta(toURI))

	matched, err := regexp.MatchString(pattern, toURI)
	if err != nil {
		return false, err
	}

	return matched, nil
}

// checkConditions 检查条件
func (rt *RoutingTable) checkConditions(conditions []string, fromURI, toURI string) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, condition := range conditions {
		// 简单的条件检查，可以根据需要扩展
		if !rt.evaluateCondition(condition, fromURI, toURI) {
			return false
		}
	}

	return true
}

// evaluateCondition 评估条件
func (rt *RoutingTable) evaluateCondition(condition, fromURI, toURI string) bool {
	// 简单的条件评估，例如：
	// "time:09:00-17:00" - 时间范围
	// "day:weekday" - 工作日
	// "user:admin" - 用户匹配
	// 可以根据需要扩展

	if strings.HasPrefix(condition, "time:") {
		// 时间条件检查
		return rt.checkTimeCondition(condition[5:])
	}

	// 默认返回true
	return true
}

// checkTimeCondition 检查时间条件
func (rt *RoutingTable) checkTimeCondition(timeRange string) bool {
	// 简化实现，实际应该解析时间范围
	// 例如 "09:00-17:00"
	return true
}

// sortRules 按优先级排序规则
func (rt *RoutingTable) sortRules() {
	// 简单的冒泡排序
	for i := 0; i < len(rt.rules)-1; i++ {
		for j := 0; j < len(rt.rules)-i-1; j++ {
			if rt.rules[j].Priority > rt.rules[j+1].Priority {
				rt.rules[j], rt.rules[j+1] = rt.rules[j+1], rt.rules[j]
			}
		}
	}
}

// GetAllRules 获取所有规则
func (rt *RoutingTable) GetAllRules() []*RoutingRule {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	result := make([]*RoutingRule, len(rt.rules))
	copy(result, rt.rules)
	return result
}

// ApplyRouting 应用路由规则
func (rt *RoutingTable) ApplyRouting(req *sip.Request) (*sip.Uri, string, error) {
	fromURI := req.From().Address.String()
	toURI := req.To().Address.String()

	rule, err := rt.FindRoute(fromURI, toURI)
	if err != nil {
		// 没有匹配的规则，返回原始目标
		addr := req.To().Address
		return &addr, "route", nil
	}

	switch rule.Action {
	case "route":
		// 路由到新目标
		var targetURI sip.Uri
		if err := sip.ParseUri(rule.Target, &targetURI); err != nil {
			return nil, "", fmt.Errorf("invalid target URI: %v", err)
		}
		return &targetURI, "route", nil

	case "reject":
		return nil, "reject", fmt.Errorf("call rejected by routing rule: %s", rule.Name)

	case "redirect":
		// 重定向
		var redirectURI sip.Uri
		if err := sip.ParseUri(rule.Target, &redirectURI); err != nil {
			return nil, "", fmt.Errorf("invalid redirect URI: %v", err)
		}
		return &redirectURI, "redirect", nil

	default:
		addr := req.To().Address
		return &addr, "route", nil
	}
}
