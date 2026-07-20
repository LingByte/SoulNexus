package workflowdef

import (
	"encoding/json"

	"github.com/LingByte/SoulNexus/internal/models"
)

// ExtractClientMeta reads _client_meta from trigger/run parameters.
// models.JSONMap is a named map type and cannot be type-asserted to map[string]interface{}.
func ExtractClientMeta(params map[string]interface{}) models.JSONMap {
	if len(params) == 0 {
		return nil
	}
	raw, ok := params["_client_meta"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case models.JSONMap:
		if len(v) == 0 {
			return nil
		}
		return v
	case map[string]interface{}:
		if len(v) == 0 {
			return nil
		}
		return models.JSONMap(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out models.JSONMap
		if err := json.Unmarshal(b, &out); err != nil || len(out) == 0 {
			return nil
		}
		return out
	}
}
