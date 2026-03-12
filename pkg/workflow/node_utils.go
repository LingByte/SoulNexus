package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func truthy(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "0" && v != "false"
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return v != nil
	}
}
