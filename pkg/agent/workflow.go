package agent

import (
	"fmt"
	"time"
)

// WorkflowBuilder 工作流构建器
type WorkflowBuilder struct {
	workflow *Workflow
}

// NewWorkflowBuilder 创建新的工作流构建器
func NewWorkflowBuilder(id, name, description string) *WorkflowBuilder {
	return &WorkflowBuilder{
		workflow: &Workflow{
			ID:          id,
			Name:        name,
			Description: description,
			Steps:       make([]WorkflowStep, 0),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
}

// AddStep 添加步骤
func (wb *WorkflowBuilder) AddStep(step WorkflowStep) *WorkflowBuilder {
	wb.workflow.Steps = append(wb.workflow.Steps, step)
	wb.workflow.UpdatedAt = time.Now()
	return wb
}

// AddSequentialStep 添加顺序步骤
func (wb *WorkflowBuilder) AddSequentialStep(stepID, agentID string, parameters map[string]interface{}) *WorkflowBuilder {
	step := WorkflowStep{
		ID:         stepID,
		AgentID:    agentID,
		Type:       StepTypeSequential,
		Parameters: parameters,
	}
	return wb.AddStep(step)
}

// AddParallelStep 添加并行步骤
func (wb *WorkflowBuilder) AddParallelStep(stepID, agentID string, parameters map[string]interface{}) *WorkflowBuilder {
	step := WorkflowStep{
		ID:         stepID,
		AgentID:    agentID,
		Type:       StepTypeParallel,
		Parameters: parameters,
	}
	return wb.AddStep(step)
}

// AddConditionalStep 添加条件步骤
func (wb *WorkflowBuilder) AddConditionalStep(stepID, agentID, condition string, parameters map[string]interface{}) *WorkflowBuilder {
	step := WorkflowStep{
		ID:         stepID,
		AgentID:    agentID,
		Type:       StepTypeConditional,
		Condition:  condition,
		Parameters: parameters,
	}
	return wb.AddStep(step)
}

// Build 构建工作流
func (wb *WorkflowBuilder) Build() *Workflow {
	return wb.workflow
}

// CreateRAGWorkflow 创建RAG工作流示例
func CreateRAGWorkflow() *Workflow {
	return NewWorkflowBuilder(
		"rag_workflow",
		"RAG Workflow",
		"检索增强生成工作流：先检索知识库，再生成回答",
	).
		AddSequentialStep("step1", "rag_agent", map[string]interface{}{
			"topK": 5,
		}).
		AddSequentialStep("step2", "llm_agent", map[string]interface{}{
			"model": "gpt-4o",
		}).
		Build()
}

// CreateMultiAgentWorkflow 创建多agent协作工作流
func CreateMultiAgentWorkflow() *Workflow {
	return NewWorkflowBuilder(
		"multi_agent_workflow",
		"Multi-Agent Workflow",
		"多agent协作工作流：图记忆 -> RAG -> LLM",
	).
		AddSequentialStep("step1", "graph_memory_agent", nil).
		AddSequentialStep("step2", "rag_agent", map[string]interface{}{
			"topK": 5,
		}).
		AddSequentialStep("step3", "llm_agent", map[string]interface{}{
			"model": "gpt-4o",
		}).
		Build()
}

// ValidateWorkflow 验证工作流
func ValidateWorkflow(workflow *Workflow, registry *Registry) error {
	if workflow.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}

	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	for i, step := range workflow.Steps {
		if step.ID == "" {
			return fmt.Errorf("step %d: step ID is required", i)
		}

		if step.AgentID == "" {
			return fmt.Errorf("step %d: agent ID is required", i)
		}

		// 验证agent是否存在
		_, err := registry.Get(step.AgentID)
		if err != nil {
			return fmt.Errorf("step %d: agent %s not found", i, step.AgentID)
		}
	}

	return nil
}
