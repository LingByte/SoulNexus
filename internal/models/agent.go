package models

import "time"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type AgentRun struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	SessionID     string    `json:"session_id" gorm:"type:varchar(64);index;not null"`
	UserID        string    `json:"user_id" gorm:"type:varchar(64);index;not null"`
	Goal          string    `json:"goal" gorm:"type:text;not null"`
	Status        string    `json:"status" gorm:"type:varchar(20);index;not null"` // queued/running/succeeded/failed/cancelled
	Phase         string    `json:"phase" gorm:"type:varchar(32);index"`           // planning/executing/reflecting
	PlanJSON      string    `json:"plan_json" gorm:"type:longtext"`
	ResultText    string    `json:"result_text" gorm:"type:longtext"`
	ErrorMessage  string    `json:"error_message" gorm:"type:text"`
	TotalSteps    int       `json:"total_steps" gorm:"default:0"`
	TotalTokens   int       `json:"total_tokens" gorm:"default:0"`
	MaxSteps      int       `json:"max_steps" gorm:"default:0"`
	MaxCostTokens int       `json:"max_cost_tokens" gorm:"default:0"`
	MaxDurationMs int64     `json:"max_duration_ms" gorm:"default:0"`
	StartedAt     time.Time `json:"started_at" gorm:"index"`
	CompletedAt   time.Time `json:"completed_at"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type AgentStep struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	RunID        string    `json:"run_id" gorm:"type:varchar(64);index;not null"`
	StepID       string    `json:"step_id" gorm:"type:varchar(64);index;not null"`
	TaskID       string    `json:"task_id" gorm:"type:varchar(64);index"`
	Title        string    `json:"title" gorm:"type:varchar(255)"`
	Instruction  string    `json:"instruction" gorm:"type:text"`
	Status       string    `json:"status" gorm:"type:varchar(20);index;not null"` // queued/running/waiting_tool/succeeded/failed/cancelled
	Model        string    `json:"model" gorm:"type:varchar(100)"`
	InputJSON    string    `json:"input_json" gorm:"type:longtext"`
	OutputText   string    `json:"output_text" gorm:"type:longtext"`
	ErrorMessage string    `json:"error_message" gorm:"type:text"`
	Feedback     string    `json:"feedback" gorm:"type:text"`
	Attempts     int       `json:"attempts" gorm:"default:0"`
	InputTokens  int       `json:"input_tokens" gorm:"default:0"`
	OutputTokens int       `json:"output_tokens" gorm:"default:0"`
	TotalTokens  int       `json:"total_tokens" gorm:"default:0"`
	LatencyMs    int64     `json:"latency_ms" gorm:"default:0"`
	StartedAt    time.Time `json:"started_at" gorm:"index"`
	CompletedAt  time.Time `json:"completed_at"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (AgentRun) TableName() string {
	return "agent_runs"
}

func (AgentStep) TableName() string {
	return "agent_steps"
}
