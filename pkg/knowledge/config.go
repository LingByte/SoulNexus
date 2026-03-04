package knowledge

import (
	"fmt"
	"strconv"
	"strings"
)

// getStringFromConfig gets string value from config map
// Supports standard key and underscore format key (for compatibility)
func getStringFromConfig(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		// Try type conversion
		return fmt.Sprintf("%v", val)
	}
	// Try underscore format (for compatibility, e.g., aliyun special requirements)
	keyWithUnderscore := strings.ReplaceAll(key, "_", "")
	if val, ok := config[keyWithUnderscore]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getIntFromConfig gets integer value from config map
func getIntFromConfig(config map[string]interface{}, key string) int {
	if config == nil {
		return 0
	}
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
			// Try using fmt.Sscanf to parse (compatible with milvus implementation)
			var i int
			if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
				return i
			}
		}
	}
	return 0
}

// getBoolFromConfig gets boolean value from config map
func getBoolFromConfig(config map[string]interface{}, key string) bool {
	if config == nil {
		return false
	}
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "1" || v == "yes"
		case int:
			return v != 0
		case float64:
			return v != 0
		}
	}
	return false
}

// mergeConfig merges two config maps, override has higher priority
func mergeConfig(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// First copy base
	for k, v := range base {
		result[k] = v
	}

	// Then override
	for k, v := range override {
		result[k] = v
	}

	return result
}

// validateRequiredConfig validates required config keys
func validateRequiredConfig(config map[string]interface{}, requiredKeys []string) error {
	for _, key := range requiredKeys {
		val := getStringFromConfig(config, key)
		if val == "" {
			return fmt.Errorf("required config key '%s' is missing or empty", key)
		}
	}
	return nil
}

// getFloatVectorFromConfig 从配置中获取float32向量
func getFloatVectorFromConfig(filter map[string]interface{}, key string) []float32 {
	if filter == nil {
		return nil
	}

	if val, ok := filter[key]; ok {
		if vec, ok := val.([]float32); ok {
			return vec
		}
		if vec, ok := val.([]interface{}); ok {
			result := make([]float32, 0, len(vec))
			for _, v := range vec {
				switch f := v.(type) {
				case float32:
					result = append(result, f)
				case float64:
					result = append(result, float32(f))
				case int:
					result = append(result, float32(f))
				}
			}
			return result
		}
	}
	return nil
}
