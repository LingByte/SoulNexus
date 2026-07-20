package workflowdef

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	runtimewf "github.com/LingByte/SoulNexus/pkg/workflow"
	"gorm.io/gorm"
)

// ServiceKnowledgeRecaller adapts knowledge.Service for workflow nodes.
type ServiceKnowledgeRecaller struct {
	db *gorm.DB
	kb *knowledge.Service
}

func NewServiceKnowledgeRecaller(db *gorm.DB, kb *knowledge.Service) *ServiceKnowledgeRecaller {
	return &ServiceKnowledgeRecaller{db: db, kb: kb}
}

func (r *ServiceKnowledgeRecaller) RecallKnowledge(
	ctx context.Context,
	groupID uint,
	namespaceID string,
	query string,
	topK int,
	minScore float64,
	outputFormat string,
) ([]map[string]interface{}, string, error) {
	if r == nil || r.kb == nil || r.db == nil {
		return nil, "", fmt.Errorf("knowledge service unavailable")
	}

	nsID, err := strconv.ParseUint(strings.TrimSpace(namespaceID), 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("invalid namespaceId: %w", err)
	}

	ns, err := knmodels.GetKnowledgeNamespaceByIDAndGroup(r.db, uint(nsID), groupID)
	if err != nil {
		return nil, "", fmt.Errorf("knowledge namespace not found: %w", err)
	}

	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	hits, err := r.kb.RecallWithOptions(ctx, ns.Namespace, query, knowledge.RecallOptions{
		TopK:     topK,
		MinScore: minScore,
	})
	if err != nil {
		return nil, "", err
	}

	payload := make([]map[string]interface{}, 0, len(hits))
	var textBlock strings.Builder
	for i, hit := range hits {
		item := knowledge.RecallHitPayload(hit)
		payload = append(payload, item)

		title, _ := item["title"].(string)
		content, _ := item["content"].(string)
		score, _ := item["score"].(float64)
		if textBlock.Len() > 0 {
			textBlock.WriteString("\n\n")
		}
		textBlock.WriteString(fmt.Sprintf("[%d] %s (score=%.3f)\n%s", i+1, title, score, content))
	}

	if len(payload) == 0 {
		return payload, "[知识库检索：未命中相关内容]", nil
	}

	return payload, textBlock.String(), nil
}

var defaultKnowledgeRecaller runtimewf.KnowledgeRecaller

// SetKnowledgeRecaller registers the global knowledge recaller for scheduled/event triggers.
func SetKnowledgeRecaller(r runtimewf.KnowledgeRecaller) {
	defaultKnowledgeRecaller = r
}

// AttachRuntimeContext sets group and knowledge recaller on a workflow context.
func AttachRuntimeContext(ctx *runtimewf.WorkflowContext, groupID uint, recaller runtimewf.KnowledgeRecaller) {
	if ctx == nil {
		return
	}
	ctx.GroupID = groupID
	if recaller != nil {
		ctx.KnowledgeRecaller = recaller
	} else {
		ctx.KnowledgeRecaller = defaultKnowledgeRecaller
	}
}
