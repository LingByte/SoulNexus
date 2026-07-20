package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
)

// registerOverviewRoutes mounts dashboard overview stats.
func (h *Handlers) registerOverviewRoutes(r *humax.Group) {
	r.GET("overview/stats", h.getOverviewDashboard)
	r.GET("overview/region-cities", h.getOverviewRegionCities)
}

const maxOverviewRangeDays = 90

func overviewDateRange(c *gin.Context) (start, end time.Time, ok bool, msgKey string) {
	loc := timeutil.Location()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	from, hasFrom := timeutil.ParseQueryDay(c.Query("from"))
	to, hasTo := timeutil.ParseQueryDay(c.Query("to"))
	if hasFrom && hasTo {
		if to.Before(from) {
			from, to = to, from
		}
		if to.After(today) {
			return time.Time{}, time.Time{}, false, i18n.KeyDateEndAfterToday
		}
		span := int(to.Sub(from).Hours()/24) + 1
		if span > maxOverviewRangeDays {
			return time.Time{}, time.Time{}, false, i18n.KeyDateRangeExceed90
		}
		return from, to, true, ""
	}
	days := 14
	if s := strings.TrimSpace(c.Query("days")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			if v > maxOverviewRangeDays {
				v = maxOverviewRangeDays
			}
			days = v
		}
	}
	end = today
	start = today.AddDate(0, 0, -(days - 1))
	return start, end, true, ""
}

// getOverviewDashboard returns a minimal overview placeholder.
func (h *Handlers) getOverviewDashboard(c *gin.Context) {
	start, end, ok, msgKey := overviewDateRange(c)
	if !ok {
		response.Render(c, response.NewI18n(response.CodeBadRequest, msgKey))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"rangeStart": start.Format("2006-01-02"),
		"rangeEnd":   end.Format("2006-01-02"),
		"calls":      0,
		"minutes":    0,
	})
}

// getOverviewRegionCities — Call geography removed; returns empty cities.
func (h *Handlers) getOverviewRegionCities(c *gin.Context) {
	start, end, ok, msgKey := overviewDateRange(c)
	if !ok {
		response.Render(c, response.NewI18n(response.CodeBadRequest, msgKey))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"rangeStart": start.Format("2006-01-02"),
		"rangeEnd":   end.Format("2006-01-02"),
		"province":   strings.TrimSpace(c.Query("province")),
		"cities":     []any{},
	})
}
