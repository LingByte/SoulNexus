package ginutil

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// QueryOptionalID reads a query param as uint. ok=false when blank; invalid=true when malformed.
func QueryOptionalID(c *gin.Context, key string) (id uint, ok bool, invalid bool) {
	s := strings.TrimSpace(c.Query(key))
	if s == "" {
		return 0, false, false
	}
	id, err := utils.ParseID(s)
	if err != nil {
		return 0, false, true
	}
	return id, true, false
}

// QueryIDList reads comma-separated decimal IDs from a query param.
// bad=true when any token is malformed.
func QueryIDList(c *gin.Context, key string) (ids []uint, bad bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, false
	}
	for _, p := range utils.ParseCommaList(raw) {
		id, err := utils.ParseID(p)
		if err != nil {
			return nil, true
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, false
	}
	return ids, false
}
