package response

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// ConfigResponse is the API response representation of a system config entry.
type ConfigResponse struct {
	ID        uint      `json:"id"`
	Key       string    `json:"key"`
	Desc      string    `json:"desc"`
	Value     string    `json:"value"`
	Format    string    `json:"format"`
	Autoload  bool      `json:"autoload"`
	Public    bool      `json:"public"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewConfigResponse converts a utils.Config model to an API response struct.
func NewConfigResponse(c utils.Config) ConfigResponse {
	return ConfigResponse{
		ID:        c.ID,
		Key:       c.Key,
		Desc:      c.Desc,
		Value:     c.Value,
		Format:    c.Format,
		Autoload:  c.Autoload,
		Public:    c.Public,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
