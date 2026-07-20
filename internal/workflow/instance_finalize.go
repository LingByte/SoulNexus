package workflowdef

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	runtimewf "github.com/LingByte/SoulNexus/pkg/workflow"
)

// InstanceRunMeta captures invocation context for a workflow run record.
type InstanceRunMeta struct {
	GroupID       uint
	UserID        uint
	TriggerUser   string
	TriggerSource string
	InputParams   models.JSONMap
	ClientMeta    models.JSONMap
}

// FinalizeInstance updates a workflow instance after execution completes.
func FinalizeInstance(inst *models.WorkflowInstance, wfCtx *runtimewf.WorkflowContext, execErr error, meta InstanceRunMeta) {
	if inst == nil {
		return
	}
	completedAt := time.Now()
	inst.CompletedAt = &completedAt
	if inst.StartedAt != nil {
		inst.DurationMs = completedAt.Sub(*inst.StartedAt).Milliseconds()
	}
	if meta.GroupID > 0 {
		inst.GroupID = meta.GroupID
	}
	inst.UserID = meta.UserID
	inst.TriggerUser = meta.TriggerUser
	inst.TriggerSource = meta.TriggerSource
	if len(meta.ClientMeta) > 0 {
		inst.ClientMeta = meta.ClientMeta
	}
	if len(meta.InputParams) > 0 {
		inst.InputParams = meta.InputParams
	}
	if execErr != nil {
		inst.Status = "failed"
		inst.ErrorMessage = execErr.Error()
		if inst.ResultData == nil {
			inst.ResultData = models.JSONMap{}
		}
		inst.ResultData["error"] = execErr.Error()
	} else {
		inst.Status = "completed"
	}
	if wfCtx != nil {
		if wfCtx.CurrentNode != "" {
			inst.CurrentNodeID = wfCtx.CurrentNode
		}
		if len(wfCtx.NodeData) > 0 {
			inst.ContextData = wfCtx.NodeData
		}
		if execErr == nil {
			inst.ResultData = models.JSONMap{
				"success": true,
				"context": wfCtx.NodeData,
			}
		}
		if len(wfCtx.Logs) > 0 {
			logs := make(models.JSONArray, 0, len(wfCtx.Logs))
			for _, entry := range wfCtx.Logs {
				logs = append(logs, map[string]interface{}{
					"timestamp": entry.Timestamp,
					"level":     entry.Level,
					"message":   entry.Message,
					"nodeId":    entry.NodeID,
					"nodeName":  entry.NodeName,
				})
			}
			inst.ExecutionLogs = logs
			inst.LogCount = len(logs)
		}
	}
}
