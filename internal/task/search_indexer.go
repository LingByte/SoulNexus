package task

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	search2 "github.com/LingByte/SoulNexus/pkg/utils/search"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var searchEngine search2.Engine
var searchIndexerRunning bool

// StartSearchIndexer starts the search indexing scheduled task
func StartSearchIndexer(db *gorm.DB, engine search2.Engine) {
	searchEngine = engine
	searchIndexerRunning = true

	c := cron.New()

	// Get scheduled task expression from configuration
	schedule := utils.GetValue(db, constants.KEY_SEARCH_INDEX_SCHEDULE)
	if schedule == "" {
		schedule = "0 */6 * * *" // Default to execute every 6 hours
	}

	// Add scheduled task
	_, err := c.AddFunc(schedule, func() {
		if !utils.GetBoolValue(db, constants.KEY_SEARCH_ENABLED) {
			logger.Info("Search is disabled, skipping index task")
			return
		}

		if err := IndexUserData(db, engine); err != nil {
			logger.Error("Search index task failed", zap.Error(err))
		} else {
			logger.Info("Search index task completed successfully")
		}
	})

	if err != nil {
		logger.Error("Failed to add search index cron job", zap.Error(err))
		return
	}

	// Start the scheduled task
	c.Start()

	logger.Info("Search indexer started", zap.String("schedule", schedule))
}

// IndexUserDataAsync asynchronously indexes user data (used at project startup)
func IndexUserDataAsync(db *gorm.DB, engine search2.Engine) {
	if engine == nil {
		logger.Warn("Search engine is nil, skipping async index")
		return
	}

	if !utils.GetBoolValue(db, constants.KEY_SEARCH_ENABLED) {
		logger.Info("Search is disabled, skipping async index")
		return
	}

	go func() {
		logger.Info("Starting async search index task on startup...")
		if err := IndexUserData(db, engine); err != nil {
			logger.Error("Async search index failed", zap.Error(err))
		} else {
			logger.Info("Async search index completed successfully")
		}
	}()
}

// IndexUserData indexes user data to the search engine
// Optimization: Process data in batches to avoid loading all data into memory at once
func IndexUserData(db *gorm.DB, engine search2.Engine) error {
	if engine == nil {
		return fmt.Errorf("search engine is nil")
	}

	ctx := context.Background()
	batchSize := utils.GetIntValue(db, constants.KEY_SEARCH_BATCH_SIZE, 100)
	totalIndexed := 0

	// Helper function: index documents in batches
	indexDocsInBatches := func(docs []search2.Doc, docType string) error {
		if len(docs) == 0 {
			return nil
		}
		// If document count is less than or equal to batch size, index directly
		if len(docs) <= batchSize {
			if err := engine.IndexBatch(ctx, docs); err != nil {
				logger.Error(fmt.Sprintf("Failed to index %s batch", docType), zap.Error(err))
				return err
			}
			return nil
		}
		// If document count is greater than batch size, index in batches
		for i := 0; i < len(docs); i += batchSize {
			end := i + batchSize
			if end > len(docs) {
				end = len(docs)
			}
			batch := docs[i:end]
			if err := engine.IndexBatch(ctx, batch); err != nil {
				logger.Error(fmt.Sprintf("Failed to index %s batch", docType), zap.Int("batch_start", i), zap.Int("batch_end", end), zap.Error(err))
				return err
			}
		}
		return nil
	}

	// 1. Index assistant data (batch processing)
	assistantOffset := 0
	assistantBatchSize := 100 // Query 100 items each time
	for {
		var assistants []models.Assistant
		if err := db.Offset(assistantOffset).Limit(assistantBatchSize).Find(&assistants).Error; err != nil {
			logger.Error("Failed to query assistants", zap.Error(err))
			break
		}
		if len(assistants) == 0 {
			break
		}

		var docs []search2.Doc
		for _, assistant := range assistants {
			doc := search2.Doc{
				ID:   fmt.Sprintf("assistant_%d", assistant.ID),
				Type: "assistant",
				Fields: map[string]interface{}{
					"userId":      strconv.Itoa(int(assistant.UserID)),
					"title":       assistant.Name,
					"description": assistant.Description,
					"content":     assistant.SystemPrompt,
					"type":        "assistant",
					"icon":        assistant.Icon,
					"url":         fmt.Sprintf("/voice-assistant/%d", assistant.ID),
					"category":    "assistant",
				},
			}
			docs = append(docs, doc)
		}

		if len(docs) > 0 {
			if err := indexDocsInBatches(docs, "assistant"); err != nil {
				logger.Error("Failed to index assistant batch", zap.Error(err))
			} else {
				totalIndexed += len(docs)
			}
		}

		if len(assistants) < assistantBatchSize {
			break
		}
		assistantOffset += assistantBatchSize
	}

	// 2. Index chat session logs (batch processing)
	chatOffset := 0
	chatBatchSize := 100
	for {
		var chatLogs []models.ChatSessionLog
		if err := db.Where("(user_message IS NOT NULL AND user_message != '') OR (agent_message IS NOT NULL AND agent_message != '')").
			Offset(chatOffset).Limit(chatBatchSize).Find(&chatLogs).Error; err != nil {
			logger.Error("Failed to query chat logs", zap.Error(err))
			break
		}
		if len(chatLogs) == 0 {
			break
		}

		var docs []search2.Doc
		for _, log := range chatLogs {
			preview := log.UserMessage
			if preview == "" {
				preview = log.AgentMessage
			}
			if len(preview) > 50 {
				preview = preview[:50]
			}

			doc := search2.Doc{
				ID:   fmt.Sprintf("chat_%d", log.ID),
				Type: "chat",
				Fields: map[string]interface{}{
					"userId":      strconv.Itoa(int(log.UserID)),
					"title":       preview,
					"description": preview,
					"content":     log.UserMessage + " " + log.AgentMessage,
					"type":        "chat",
					"url":         fmt.Sprintf("/voice-assistant/%d", log.AssistantID),
					"category":    "chat",
				},
			}
			docs = append(docs, doc)
		}

		if len(docs) > 0 {
			if err := indexDocsInBatches(docs, "chat"); err != nil {
				logger.Error("Failed to index chat batch", zap.Error(err))
			} else {
				totalIndexed += len(docs)
			}
		}

		if len(chatLogs) < chatBatchSize {
			break
		}
		chatOffset += chatBatchSize
	}

	// 3. Index notifications (batch processing)
	notificationOffset := 0
	notificationBatchSize := 100
	for {
		var notifications []notification.InternalNotification
		if err := db.Where("title IS NOT NULL AND title != ''").
			Offset(notificationOffset).Limit(notificationBatchSize).Find(&notifications).Error; err != nil {
			logger.Error("Failed to query notifications", zap.Error(err))
			break
		}
		if len(notifications) == 0 {
			break
		}

		var docs []search2.Doc
		for _, notification := range notifications {
			doc := search2.Doc{
				ID:   fmt.Sprintf("notification_%d", notification.ID),
				Type: "notification",
				Fields: map[string]interface{}{
					"userId":      strconv.Itoa(int(notification.UserID)),
					"title":       notification.Title,
					"description": notification.Content,
					"content":     notification.Title + " " + notification.Content,
					"type":        "notification",
					"url":         "/notification",
					"category":    "notification",
				},
			}
			docs = append(docs, doc)
		}

		if len(docs) > 0 {
			if err := indexDocsInBatches(docs, "notification"); err != nil {
				logger.Error("Failed to index notification batch", zap.Error(err))
			} else {
				totalIndexed += len(docs)
			}
		}

		if len(notifications) < notificationBatchSize {
			break
		}
		notificationOffset += notificationBatchSize
	}

	logger.Info("Indexed documents completed", zap.Int("total_count", totalIndexed))
	return nil
}
