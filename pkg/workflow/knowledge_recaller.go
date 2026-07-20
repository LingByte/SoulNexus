package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import "context"

// KnowledgeRecaller performs knowledge-base recall for workflow nodes.
type KnowledgeRecaller interface {
	RecallKnowledge(
		ctx context.Context,
		groupID uint,
		namespaceID string,
		query string,
		topK int,
		minScore float64,
		outputFormat string,
	) (hits []map[string]interface{}, textBlock string, err error)
}
