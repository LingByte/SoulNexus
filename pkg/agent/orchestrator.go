package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Orchestrator Agent协调器，负责任务分发和路由
type Orchestrator struct {
	registry       *Registry
	decisionEngine *DecisionEngine
	logger         *zap.Logger
	mu             sync.RWMutex
	tasks          map[string]*TaskRequest
	results        map[string]*TaskResponse
}

// NewOrchestrator 创建新的协调器
func NewOrchestrator(registry *Registry, logger *zap.Logger) *Orchestrator {
	// 创建决策引擎
	decisionEngine := NewDecisionEngine(registry, logger, nil)

	return &Orchestrator{
		registry:       registry,
		decisionEngine: decisionEngine,
		logger:         logger,
		tasks:          make(map[string]*TaskRequest),
		results:        make(map[string]*TaskResponse),
	}
}

// Process 处理任务请求
func (o *Orchestrator) Process(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	startTime := time.Now()

	// 设置任务ID（如果没有）
	if request.ID == "" {
		request.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	// 设置创建时间
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now()
	}

	o.mu.Lock()
	o.tasks[request.ID] = request
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		delete(o.tasks, request.ID)
		o.mu.Unlock()
	}()

	o.logger.Info("Processing task",
		zap.String("taskID", request.ID),
		zap.String("type", request.Type),
	)

	// 使用决策引擎选择Agents
	selectedAgents, err := o.decisionEngine.Decide(ctx, request)
	if err != nil {
		return o.createErrorResponse(request.ID, fmt.Sprintf("Failed to decide agents: %v", err)), nil
	}

	// 根据选择的Agent数量决定执行方式
	var response *TaskResponse
	if len(selectedAgents) == 1 {
		// 单Agent执行
		response, err = o.processSingleAgent(ctx, request, selectedAgents[0], startTime)
	} else if len(selectedAgents) > 1 {
		// 多Agent执行（并行或链式）
		// 检查是否需要链式执行
		if o.needsChainExecution(request, selectedAgents) {
			response, err = o.processChainAgents(ctx, request, selectedAgents, startTime)
		} else {
			response, err = o.processParallelAgents(ctx, request, selectedAgents, startTime)
		}
	} else {
		return o.createErrorResponse(request.ID, "No agents selected"), nil
	}

	if err != nil {
		return o.createErrorResponse(request.ID, fmt.Sprintf("Processing failed: %v", err)), nil
	}

	// 保存结果
	o.mu.Lock()
	o.results[request.ID] = response
	o.mu.Unlock()

	// 记录执行历史
	if response.AgentID != "" {
		o.decisionEngine.RecordExecution(ExecutionRecord{
			TaskID:    request.ID,
			AgentID:   response.AgentID,
			TaskType:  request.Type,
			Success:   response.Success,
			Duration:  response.ProcessingTime,
			Timestamp: time.Now(),
			Error:     response.Error,
		})
	}

	o.logger.Info("Task completed",
		zap.String("taskID", request.ID),
		zap.String("agentID", response.AgentID),
		zap.Duration("processingTime", response.ProcessingTime),
		zap.Bool("success", response.Success),
	)

	return response, nil
}

// processSingleAgent 处理单Agent任务
func (o *Orchestrator) processSingleAgent(ctx context.Context, request *TaskRequest, agent Agent, startTime time.Time) (*TaskResponse, error) {
	o.updateAgentStatus(agent.ID(), AgentStatusBusy, 1)
	defer o.updateAgentStatus(agent.ID(), AgentStatusIdle, -1)

	response, err := agent.Process(ctx, request)
	if err != nil {
		return o.createErrorResponse(request.ID, fmt.Sprintf("Agent processing failed: %v", err)), nil
	}

	response.ID = request.ID
	response.AgentID = agent.ID()
	response.ProcessingTime = time.Since(startTime)
	response.CreatedAt = time.Now()

	return response, nil
}

// processParallelAgents 并行处理多Agent任务
func (o *Orchestrator) processParallelAgents(ctx context.Context, request *TaskRequest, agents []Agent, startTime time.Time) (*TaskResponse, error) {
	var wg sync.WaitGroup
	responses := make([]*TaskResponse, len(agents))
	errors := make([]error, len(agents))

	for i, agent := range agents {
		wg.Add(1)
		go func(idx int, ag Agent) {
			defer wg.Done()
			o.updateAgentStatus(ag.ID(), AgentStatusBusy, 1)
			defer o.updateAgentStatus(ag.ID(), AgentStatusIdle, -1)

			resp, err := ag.Process(ctx, request)
			if err != nil {
				errors[idx] = err
				responses[idx] = o.createErrorResponse(request.ID, err.Error())
			} else {
				responses[idx] = resp
			}
		}(i, agent)
	}

	wg.Wait()

	// 合并结果（选择最成功的响应，或合并内容）
	bestResponse := responses[0]
	for _, resp := range responses[1:] {
		if resp.Success && !bestResponse.Success {
			bestResponse = resp
		} else if resp.Success && bestResponse.Success {
			// 如果都成功，合并内容
			bestResponse.Content += "\n\n" + resp.Content
		}
	}

	bestResponse.ID = request.ID
	bestResponse.ProcessingTime = time.Since(startTime)
	bestResponse.CreatedAt = time.Now()

	return bestResponse, nil
}

// processChainAgents 链式处理多Agent任务
func (o *Orchestrator) processChainAgents(ctx context.Context, request *TaskRequest, agents []Agent, startTime time.Time) (*TaskResponse, error) {
	currentRequest := request
	var finalResponse *TaskResponse

	for i, agent := range agents {
		o.updateAgentStatus(agent.ID(), AgentStatusBusy, 1)

		response, err := agent.Process(ctx, currentRequest)
		if err != nil {
			o.updateAgentStatus(agent.ID(), AgentStatusIdle, -1)
			return o.createErrorResponse(request.ID, fmt.Sprintf("Agent %s failed: %v", agent.ID(), err)), nil
		}

		o.updateAgentStatus(agent.ID(), AgentStatusIdle, -1)

		// 更新请求内容（用于下一个Agent）
		if response.Success && response.Content != "" {
			currentRequest.Content = response.Content
		}

		finalResponse = response
		o.logger.Debug("Chain step completed",
			zap.String("taskID", request.ID),
			zap.Int("step", i+1),
			zap.String("agentID", agent.ID()),
		)
	}

	if finalResponse != nil {
		finalResponse.ID = request.ID
		finalResponse.ProcessingTime = time.Since(startTime)
		finalResponse.CreatedAt = time.Now()
	}

	return finalResponse, nil
}

// needsChainExecution 判断是否需要链式执行
func (o *Orchestrator) needsChainExecution(request *TaskRequest, agents []Agent) bool {
	// 如果请求中明确指定了链式执行
	if strategy, ok := request.Parameters["strategy"].(string); ok && strategy == "chain" {
		return true
	}

	// 如果Agent类型暗示需要链式执行（如：graph -> rag -> llm）
	if len(agents) > 1 {
		agentTypes := make([]string, len(agents))
		for i, agent := range agents {
			agentTypes[i] = agent.ID()
		}

		// 检查是否是典型的链式模式
		if contains(agentTypes, "graph_memory_agent") &&
			contains(agentTypes, "rag_agent") &&
			contains(agentTypes, "llm_agent") {
			return true
		}
	}

	return false
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ProcessWorkflow 处理工作流
func (o *Orchestrator) ProcessWorkflow(ctx context.Context, workflow *Workflow, initialRequest *TaskRequest) (*TaskResponse, error) {
	startTime := time.Now()

	o.logger.Info("Processing workflow",
		zap.String("workflowID", workflow.ID),
		zap.String("workflowName", workflow.Name),
		zap.Int("steps", len(workflow.Steps)),
	)

	// 构建步骤执行计划
	executionPlan, err := o.buildExecutionPlan(workflow)
	if err != nil {
		return o.createErrorResponse(initialRequest.ID, fmt.Sprintf("Failed to build execution plan: %v", err)), nil
	}

	// 执行工作流
	results := make(map[string]*TaskResponse)
	currentRequest := initialRequest

	for _, stepGroup := range executionPlan {
		// 并行执行组内的步骤
		var wg sync.WaitGroup
		stepResults := make(map[string]*TaskResponse)
		var stepMu sync.Mutex

		for _, step := range stepGroup {
			wg.Add(1)
			go func(s WorkflowStep) {
				defer wg.Done()

				// 构建步骤请求
				stepRequest := &TaskRequest{
					ID:         fmt.Sprintf("%s_step_%s", initialRequest.ID, s.ID),
					Type:       currentRequest.Type,
					Content:    currentRequest.Content,
					Context:    currentRequest.Context,
					Parameters: mergeMaps(currentRequest.Parameters, s.Parameters),
					Metadata: mergeMaps(currentRequest.Metadata, map[string]interface{}{
						"workflowID": workflow.ID,
						"stepID":     s.ID,
					}),
					CreatedAt: time.Now(),
				}

				// 处理步骤
				response, err := o.processStep(ctx, s, stepRequest, results)
				if err != nil {
					o.logger.Error("Step processing failed",
						zap.String("stepID", s.ID),
						zap.Error(err),
					)
					response = o.createErrorResponse(stepRequest.ID, err.Error())
				}

				stepMu.Lock()
				stepResults[s.ID] = response
				stepMu.Unlock()
			}(step)
		}

		wg.Wait()

		// 合并步骤结果
		for stepID, result := range stepResults {
			results[stepID] = result
			// 更新当前请求内容（用于后续步骤）
			if result.Success && result.Content != "" {
				currentRequest.Content = result.Content
			}
		}
	}

	// 构建最终响应
	finalResponse := &TaskResponse{
		ID:             initialRequest.ID,
		Success:        true,
		Content:        currentRequest.Content,
		Data:           map[string]interface{}{"workflowResults": results},
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
	}

	// 检查是否有失败的步骤
	for _, result := range results {
		if !result.Success {
			finalResponse.Success = false
			finalResponse.Error = "Some workflow steps failed"
			break
		}
	}

	return finalResponse, nil
}

// processStep 处理单个工作流步骤
func (o *Orchestrator) processStep(ctx context.Context, step WorkflowStep, request *TaskRequest, previousResults map[string]*TaskResponse) (*TaskResponse, error) {
	// 检查条件步骤
	if step.Type == StepTypeConditional && step.Condition != "" {
		// 这里可以实现条件判断逻辑
		// 简化实现：如果条件包含"skip"，则跳过
		if step.Condition == "skip" {
			return &TaskResponse{
				ID:      request.ID,
				Success: true,
				Content: "Step skipped",
			}, nil
		}
	}

	// 获取agent
	agent, err := o.registry.Get(step.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %s", step.AgentID)
	}

	// 处理任务
	return agent.Process(ctx, request)
}

// buildExecutionPlan 构建执行计划（处理依赖关系）
func (o *Orchestrator) buildExecutionPlan(workflow *Workflow) ([][]WorkflowStep, error) {
	// 简化实现：按顺序执行
	// 实际应该使用拓扑排序处理依赖关系
	plan := make([][]WorkflowStep, 0)

	for _, step := range workflow.Steps {
		if step.Type == StepTypeParallel {
			// 并行步骤放在同一组
			if len(plan) == 0 {
				plan = append(plan, []WorkflowStep{})
			}
			plan[len(plan)-1] = append(plan[len(plan)-1], step)
		} else {
			// 顺序步骤
			plan = append(plan, []WorkflowStep{step})
		}
	}

	return plan, nil
}

// updateAgentStatus 更新agent状态
func (o *Orchestrator) updateAgentStatus(agentID string, status string, taskDelta int) {
	agentStatus, err := o.registry.GetStatus(agentID)
	if err != nil {
		return
	}

	agentStatus.Status = status
	agentStatus.ActiveTasks += taskDelta
	if agentStatus.ActiveTasks < 0 {
		agentStatus.ActiveTasks = 0
	}
	agentStatus.TotalTasks++
	agentStatus.LastActivity = time.Now()

	o.registry.UpdateStatus(agentID, agentStatus)
}

// createErrorResponse 创建错误响应
func (o *Orchestrator) createErrorResponse(taskID, errorMsg string) *TaskResponse {
	return &TaskResponse{
		ID:        taskID,
		Success:   false,
		Error:     errorMsg,
		CreatedAt: time.Now(),
	}
}

// GetTaskStatus 获取任务状态
func (o *Orchestrator) GetTaskStatus(taskID string) (*TaskResponse, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result, exists := o.results[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return result, nil
}

// mergeMaps 合并两个map
func mergeMaps(m1, m2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}
