package sessions

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"unicode"

	"github.com/LingByte/SoulNexus/pkg/voice/constants"
	"go.uber.org/zap"
)

// FilterWordComponent filter check stage
type FilterWordComponent struct {
	logger      *zap.Logger
	filterWords map[string]bool
}

func (s *FilterWordComponent) Name() string {
	return constants.COMPONENT_FILTER_WORD
}

func (s *FilterWordComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	text, ok := data.(string)
	if !ok {
		return nil, false, fmt.Errorf("invalid data type")
	}

	// 检测是否包含emoji
	if s.containsEmoji(text) {
		s.logger.Info("[FilterWord] 文本包含emoji，被过滤", zap.String("text", text))
		return nil, false, nil // 中断管道
	}

	s.logger.Debug("[FilterWord] 文本通过", zap.String("text", text))
	return text, true, nil
}

// containsEmoji 检测文本是否包含emoji
func (s *FilterWordComponent) containsEmoji(text string) bool {
	for _, r := range text {
		// Emoji通常在以下Unicode范围内：
		// 1. Emoticons: U+1F600 - U+1F64F
		// 2. Miscellaneous Symbols and Pictographs: U+1F300 - U+1F5FF
		// 3. Transport and Map Symbols: U+1F680 - U+1F6FF
		// 4. Supplemental Symbols and Pictographs: U+1F900 - U+1F9FF
		// 5. Symbols and Pictographs Extended-A: U+1FA70 - U+1FAFF
		// 6. Miscellaneous Symbols: U+2600 - U+26FF
		// 7. Dingbats: U+2700 - U+27BF
		if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
			(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
			(r >= 0x1FA70 && r <= 0x1FAFF) || // Symbols and Pictographs Extended-A
			(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
			(r >= 0x2700 && r <= 0x27BF) || // Dingbats
			(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
			(r >= 0x1F1E6 && r <= 0x1F1FF) { // Regional Indicator Symbols
			return true
		}

		// 检查是否为Emoji Presentation
		if unicode.Is(unicode.So, r) || unicode.Is(unicode.Sk, r) {
			// 某些符号可能是emoji
			if r > 0x2000 {
				return true
			}
		}
	}
	return false
}
