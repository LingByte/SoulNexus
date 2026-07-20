package voiceprintsvc

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/voiceprint"
	"github.com/gin-gonic/gin"
)

// Options configures the embedded voiceprint HTTP microservice.
type Options struct {
	APIKey string
}

// RegisterRoutes mounts /voiceprint/* endpoints expected by lingllm/voiceprint HTTP client.
func RegisterRoutes(r gin.IRouter, opts Options) error {
	store, err := NewStore(strings.TrimSpace(utils.GetEnv("VOICEPRINT_DATA_DIR")))
	if err != nil {
		return err
	}
	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(utils.GetEnv("VOICEPRINT_API_KEY"))
	}

	g := r.Group("/voiceprint")
	g.Use(requireAPIKey(apiKey))
	{
		g.GET("/health", func(c *gin.Context) { voiceprintHealth(c, store) })
		g.POST("/register", func(c *gin.Context) { voiceprintRegister(c, store) })
		g.POST("/identify", func(c *gin.Context) { voiceprintIdentify(c, store) })
		g.DELETE("/:speakerID", func(c *gin.Context) { voiceprintDelete(c, store) })
	}
	return nil
}

func requireAPIKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			c.Next()
			return
		}
		got := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(got), "bearer ") {
			got = strings.TrimSpace(got[7:])
		}
		if got == "" {
			got = strings.TrimSpace(c.Query("key"))
		}
		if got != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"msg":     "invalid api key",
			})
			return
		}
		c.Next()
	}
}

func voiceprintHealth(c *gin.Context, store *Store) {
	total, err := store.CountAll()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":            "degraded",
			"total_voiceprints": 0,
			"msg":               err.Error(),
			"timestamp":         time.Now(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":            "ok",
		"total_voiceprints": total,
		"timestamp":         time.Now(),
	})
}

func voiceprintRegister(c *gin.Context, store *Store) {
	speakerID := strings.TrimSpace(c.PostForm("speaker_id"))
	agentID := strings.TrimSpace(c.PostForm("agent_id"))
	if speakerID == "" || agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "speaker_id and agent_id are required"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "audio file required"})
		return
	}
	fh, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "open audio failed"})
		return
	}
	defer fh.Close()
	audio, err := io.ReadAll(io.LimitReader(fh, 8<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "read audio failed"})
		return
	}
	if err := voiceprint.ValidateWAVFormat(audio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if err := store.Save(agentID, speakerID, audio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"msg":        "registered",
		"speaker_id": speakerID,
		"timestamp":  time.Now(),
	})
}

func voiceprintIdentify(c *gin.Context, store *Store) {
	agentID := strings.TrimSpace(c.PostForm("agent_id"))
	rawIDs := strings.TrimSpace(c.PostForm("speaker_ids"))
	if agentID == "" || rawIDs == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "agent_id and speaker_ids are required"})
		return
	}
	candidates := make([]string, 0)
	for _, id := range strings.Split(rawIDs, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "speaker_ids is empty"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "audio file required"})
		return
	}
	fh, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "open audio failed"})
		return
	}
	defer fh.Close()
	queryAudio, err := io.ReadAll(io.LimitReader(fh, 8<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "read audio failed"})
		return
	}
	if err := voiceprint.ValidateWAVFormat(queryAudio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}

	bestID := ""
	bestScore := -1.0
	for _, speakerID := range candidates {
		enrolled, err := store.Load(agentID, speakerID)
		if err != nil {
			continue
		}
		score, err := compareWAVScore(queryAudio, enrolled)
		if err != nil {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestID = speakerID
		}
	}
	if bestID == "" || bestScore < 0 {
		c.JSON(http.StatusOK, gin.H{
			"speaker_id": "",
			"score":      0,
			"timestamp":  time.Now(),
		})
		return
	}
	confidence := "low"
	switch {
	case bestScore >= 0.8:
		confidence = "very_high"
	case bestScore >= 0.6:
		confidence = "high"
	case bestScore >= 0.4:
		confidence = "medium"
	}
	c.JSON(http.StatusOK, gin.H{
		"speaker_id": bestID,
		"score":      bestScore,
		"confidence": confidence,
		"timestamp":  time.Now(),
	})
}

func voiceprintDelete(c *gin.Context, store *Store) {
	speakerID := strings.TrimSpace(c.Param("speakerID"))
	agentID := strings.TrimSpace(c.PostForm("agent_id"))
	if speakerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "speaker_id is required"})
		return
	}
	if agentID == "" {
		agentID = strings.TrimSpace(c.Query("agent_id"))
	}
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "agent_id is required"})
		return
	}
	if err := store.Delete(agentID, speakerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"msg":        "deleted",
		"speaker_id": speakerID,
		"timestamp":  time.Now(),
	})
}
