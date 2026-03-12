package knowledge

import (
	"fmt"
	"strings"
)

// GetStringFromConfig gets string value from config map
func GetStringFromConfig(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	keyWithUnderscore := strings.ReplaceAll(key, "_", "")
	if val, ok := config[keyWithUnderscore]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
