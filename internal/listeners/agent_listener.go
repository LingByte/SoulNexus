package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func InitAgentListener() {
	utils.Sig().Connect(constants.AgentCreate, func(sender any, params ...any) {
		user, ok := sender.(*models.User)
		if !ok {
			return
		}

		db, ok := params[0].(*gorm.DB)
		if !ok {
			return
		}

		agent, ok := params[1].(*models.Agent)
		if !ok {
			return
		}
		logger.Info("user created an agent", zap.Uint("userId", user.ID), zap.String("agentName", agent.Name))

		title := "New Agent Created Successfully"
		content := fmt.Sprintf("Dear %s, you have successfully created a new AI agent:\n\n"+
			"Agent Name: %s\n"+
			"Agent Description: %s\n\n"+
			"You can further configure the agent on the agent management page.\n"+
			"Go to the agent management page now to start using it!",
			user.EffectiveDisplayName(), agent.Name, agent.Description)

		notification.NewInternalNotificationService(db).Send(user.ID, title, content)
	})
}
