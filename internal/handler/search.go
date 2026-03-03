package handlers

import (
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetSearchStatus gets search function status
func (h *Handlers) GetSearchStatus(c *gin.Context) {
	enabled := utils.GetBoolValue(h.db, constants.KEY_SEARCH_ENABLED)
	searchPath := utils.GetValue(h.db, constants.KEY_SEARCH_PATH)
	batchSize := utils.GetIntValue(h.db, constants.KEY_SEARCH_BATCH_SIZE, 100)
	schedule := utils.GetValue(h.db, constants.KEY_SEARCH_INDEX_SCHEDULE)

	response.Success(c, "Get search status", gin.H{
		"enabled":    enabled,
		"searchPath": searchPath,
		"batchSize":  batchSize,
		"schedule":   schedule,
	})
}

// UpdateSearchConfig updates search configuration
func (h *Handlers) UpdateSearchConfig(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	var input struct {
		Enabled   *bool   `json:"enabled"`
		Path      *string `json:"path"`
		BatchSize *int    `json:"batchSize"`
		Schedule  *string `json:"schedule"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "invalid request", "Invalid parameters")
		return
	}

	if input.Enabled != nil {
		utils.SetValue(h.db, constants.KEY_SEARCH_ENABLED, strconv.FormatBool(*input.Enabled), "bool", true, true)
	}

	if input.Path != nil {
		utils.SetValue(h.db, constants.KEY_SEARCH_PATH, *input.Path, "text", true, false)
	}

	if input.BatchSize != nil {
		utils.SetValue(h.db, constants.KEY_SEARCH_BATCH_SIZE, strconv.Itoa(*input.BatchSize), "int", true, false)
	}

	if input.Schedule != nil {
		utils.SetValue(h.db, constants.KEY_SEARCH_INDEX_SCHEDULE, *input.Schedule, "text", true, false)
	}

	// Reload configuration
	utils.LoadAutoloads(h.db)

	response.Success(c, "Update successful", nil)
}

// EnableSearch enables search function
func (h *Handlers) EnableSearch(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	utils.SetValue(h.db, constants.KEY_SEARCH_ENABLED, "true", "bool", true, true)
	utils.LoadAutoloads(h.db)

	response.Success(c, "Search function enabled", nil)
}

// DisableSearch disables search function
func (h *Handlers) DisableSearch(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	utils.SetValue(h.db, constants.KEY_SEARCH_ENABLED, "false", "bool", true, true)
	utils.LoadAutoloads(h.db)

	response.Success(c, "Search function disabled", nil)
}
