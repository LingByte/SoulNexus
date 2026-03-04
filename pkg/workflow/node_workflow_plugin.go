package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"gorm.io/gorm"
)

// WorkflowPluginNode 工作流插件节点
type WorkflowPluginNode struct {
	Node
	PluginID   uint                   // 插件ID
	Plugin     *models.WorkflowPlugin // 插件定义（运行时加载）
	Parameters map[string]interface{} // 用户配置的参数
}

func (w *WorkflowPluginNode) Base() *Node {
	return &w.Node
}

func (w *WorkflowPluginNode) Run(ctx *WorkflowContext) ([]string, error) {
	// 如果插件信息还没有加载，从数据库加载
	if w.Plugin == nil {
		if err := w.loadPlugin(ctx.db); err != nil {
			return nil, fmt.Errorf("加载工作流插件失败: %v", err)
		}
	}

	// 准备输入数据
	// 使用 PrepareInputs 方法从上下文中获取输入数据
	// 这会根据 InputParams 的映射关系正确解析数据
	inputs, err := w.Node.PrepareInputs(ctx)
	if err != nil {
		ctx.AddLog("error", fmt.Sprintf("准备输入参数失败: %v", err), w.ID, w.Name)
		return nil, err
	}

	// 记录输入参数
	if len(inputs) > 0 {
		inputJSON, _ := json.Marshal(inputs)
		ctx.AddLog("info", fmt.Sprintf("Input: %s", string(inputJSON)), w.ID, w.Name)
	}

	// 执行工作流插件（实际上是执行插件对应的工作流）
	result, err := w.executeWorkflowPlugin(ctx, inputs)
	if err != nil {
		ctx.AddLog("error", fmt.Sprintf("工作流插件执行失败: %v", err), w.ID, w.Name)
		return nil, err
	}

	// 记录输出参数
	if len(result) > 0 {
		outputJSON, _ := json.Marshal(result)
		ctx.AddLog("info", fmt.Sprintf("Output: %s", string(outputJSON)), w.ID, w.Name)
	}

	// 使用 PersistOutputs 方法将输出存储到上下文中
	// 这会根据 OutputParams 的映射关系正确存储数据
	w.Node.PersistOutputs(ctx, result)

	ctx.AddLog("success", "工作流插件执行完成", w.ID, w.Name)
	return w.NextNodes, nil
}

// loadPlugin 从数据库加载插件信息
func (w *WorkflowPluginNode) loadPlugin(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}

	var plugin models.WorkflowPlugin
	if err := db.First(&plugin, w.PluginID).Error; err != nil {
		return fmt.Errorf("加载工作流插件失败: %v", err)
	}

	w.Plugin = &plugin
	return nil
}

// executeWorkflowPlugin 执行工作流插件
func (w *WorkflowPluginNode) executeWorkflowPlugin(ctx *WorkflowContext, inputs map[string]interface{}) (map[string]interface{}, error) {
	// 工作流快照已经是 WorkflowGraph 类型，不需要解析
	workflowGraph := w.Plugin.WorkflowSnapshot

	ctx.AddLog("info", fmt.Sprintf("执行工作流插件: %s", w.Plugin.DisplayName), w.ID, w.Name)

	// 创建子工作流定义
	subWorkflowDef := &models.WorkflowDefinition{
		ID:         w.Plugin.ID, // 使用插件ID作为工作流ID
		Name:       w.Plugin.DisplayName,
		Definition: workflowGraph,
	}

	// 构建子工作流运行时实例
	subWorkflow, err := w.buildSubWorkflow(subWorkflowDef, ctx.db)
	if err != nil {
		return nil, fmt.Errorf("构建子工作流失败: %v", err)
	}

	// 设置子工作流的输入参数
	if subWorkflow.Context == nil {
		subWorkflow.Context = NewWorkflowContextWithDB(fmt.Sprintf("subworkflow-%d", w.Plugin.ID), ctx.db)
	}

	// 将输入参数传递给子工作流
	if subWorkflow.Context.Parameters == nil {
		subWorkflow.Context.Parameters = make(map[string]interface{})
	}
	for k, v := range inputs {
		subWorkflow.Context.Parameters[k] = v
	}

	// 同时将输入参数存储到 NodeData 中，以支持子工作流中引用父工作流开始节点的参数
	// 这是为了处理子工作流的开始节点的 inputMap 中可能引用父工作流开始节点 ID 的情况
	if subWorkflow.Context.NodeData == nil {
		subWorkflow.Context.NodeData = make(map[string]interface{})
	}

	// 获取父工作流的开始节点 ID（从当前上下文中推断）
	// 实际上，我们需要从父工作流中获取开始节点 ID
	// 但这里我们没有父工作流的引用，所以我们只能尝试从 ctx 中推断
	// 一个更好的方案是在工作流插件节点中传递父工作流的开始节点 ID

	// 为了支持子工作流中引用父工作流开始节点的参数，我们需要在 NodeData 中创建对应的结构
	// 例如：如果子工作流的开始节点的 inputMap 是 {"city": "start-xxx.city"}
	// 我们需要在 NodeData 中创建 {"start-xxx": {"city": value}}

	// 但我们不知道父工作流的开始节点 ID，所以我们需要从工作流插件节点中获取
	// 这需要修改工作流插件节点的构造或执行逻辑

	// 设置日志转发 - 将子工作流的日志转发到父工作流
	if ctx.LogSender != nil {
		subWorkflow.Context.LogSender = &SubWorkflowLogForwarder{
			parentSender:   ctx.LogSender,
			parentNodeID:   w.ID,
			parentNodeName: w.Name,
		}
	}

	ctx.AddLog("info", fmt.Sprintf("开始执行子工作流: %s", w.Plugin.DisplayName), w.ID, w.Name)

	// 执行子工作流
	execErr := subWorkflow.Execute()
	if execErr != nil {
		ctx.AddLog("error", fmt.Sprintf("子工作流执行失败: %v", execErr), w.ID, w.Name)
		return nil, fmt.Errorf("子工作流执行失败: %v", execErr)
	}

	ctx.AddLog("success", fmt.Sprintf("子工作流执行完成: %s", w.Plugin.DisplayName), w.ID, w.Name)

	// 收集子工作流的输出结果
	result := make(map[string]interface{})

	// 从子工作流的上下文中提取结果
	if subWorkflow.Context != nil && subWorkflow.Context.NodeData != nil {
		// 首先尝试从 workflow_result 中获取结果（这是结束节点设置的）
		if workflowResult, exists := subWorkflow.Context.NodeData["workflow_result"]; exists {
			if resultMap, ok := workflowResult.(map[string]interface{}); ok {
				result = resultMap
			}
		}

		// 如果没有找到 workflow_result，尝试从输出参数定义中提取
		if len(result) == 0 && w.Plugin.OutputSchema.Parameters != nil {
			for _, param := range w.Plugin.OutputSchema.Parameters {
				// 尝试从子工作流的结果中获取对应的输出
				if value, exists := subWorkflow.Context.NodeData[param.Name]; exists {
					result[param.Name] = value
				}
			}
		}

		// 如果仍然没有找到结果，返回所有 NodeData
		if len(result) == 0 {
			result = subWorkflow.Context.NodeData
		}
	}

	return result, nil
}

// SubWorkflowLogForwarder 子工作流日志转发器
type SubWorkflowLogForwarder struct {
	parentSender   LogSender
	parentNodeID   string
	parentNodeName string
}

func (s *SubWorkflowLogForwarder) SendLog(log ExecutionLog) error {
	if s.parentSender == nil {
		return nil
	}

	// 修改日志，添加子工作流前缀，但保持原始的节点信息
	// 只在必要时添加前缀，避免过度嵌套
	modifiedLog := ExecutionLog{
		Timestamp: log.Timestamp,
		Level:     log.Level,
		Message:   log.Message, // 不添加重复的前缀
		NodeID:    log.NodeID,
		NodeName:  fmt.Sprintf("[%s] %s", s.parentNodeName, log.NodeName), // 简化格式
	}

	return s.parentSender.SendLog(modifiedLog)
}

// buildSubWorkflow 构建子工作流
func (w *WorkflowPluginNode) buildSubWorkflow(def *models.WorkflowDefinition, db *gorm.DB) (*Workflow, error) {
	if def == nil {
		return nil, fmt.Errorf("workflow definition is nil")
	}
	if len(def.Definition.Nodes) == 0 {
		return nil, fmt.Errorf("workflow definition %s has no nodes", def.Name)
	}

	wf := NewWorkflow(fmt.Sprintf("subworkflow-%d", def.ID))
	if db != nil {
		wf.Context = NewWorkflowContextWithDB(fmt.Sprintf("subworkflow-%d", def.ID), db)
	} else {
		wf.Context = NewWorkflowContext(fmt.Sprintf("subworkflow-%d", def.ID))
	}

	nodeRegistry := make(map[string]ExecutableNode, len(def.Definition.Nodes))
	startCount := 0
	endCount := 0

	// 构建节点
	for _, nodeSchema := range def.Definition.Nodes {
		if nodeSchema.ID == "" {
			return nil, fmt.Errorf("node id cannot be empty")
		}
		if _, exists := nodeRegistry[nodeSchema.ID]; exists {
			return nil, fmt.Errorf("duplicate node id %s", nodeSchema.ID)
		}

		baseNode := Node{
			ID:           nodeSchema.ID,
			Name:         nodeSchema.Name,
			Type:         NodeType(nodeSchema.Type),
			InputParams:  w.toNativeMap(nodeSchema.InputMap),
			OutputParams: w.toNativeMap(nodeSchema.OutputMap),
			Properties:   w.toNativeMap(nodeSchema.Properties),
		}

		execNode, err := w.instantiateNode(baseNode)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", nodeSchema.ID, err)
		}

		if baseNode.Type == NodeTypeStart {
			startCount++
		}
		if baseNode.Type == NodeTypeEnd {
			endCount++
		}
		nodeRegistry[baseNode.ID] = execNode
	}

	if startCount != 1 {
		return nil, fmt.Errorf("workflow requires exactly one start node, got %d", startCount)
	}
	if endCount == 0 {
		return nil, fmt.Errorf("workflow requires at least one end node")
	}

	// 构建节点连接关系
	successors := make(map[string][]string)
	for _, edge := range def.Definition.Edges {
		if edge.Source == "" || edge.Target == "" {
			continue
		}
		if _, ok := nodeRegistry[edge.Source]; !ok {
			return nil, fmt.Errorf("edge references unknown source node %s", edge.Source)
		}
		if _, ok := nodeRegistry[edge.Target]; !ok {
			return nil, fmt.Errorf("edge references unknown target node %s", edge.Target)
		}
		successors[edge.Source] = append(successors[edge.Source], edge.Target)
	}

	var startID, endID string
	for id, node := range nodeRegistry {
		if node.Base() == nil {
			continue
		}
		if next, ok := successors[id]; ok {
			node.Base().NextNodes = next
		}
		switch node.Base().Type {
		case NodeTypeStart:
			if startID == "" {
				startID = id
			}
		case NodeTypeEnd:
			if endID == "" {
				endID = id
			}
		}
		wf.RegisterNode(node)
	}

	if startID == "" {
		return nil, fmt.Errorf("workflow definition %s missing start node", def.Name)
	}
	wf.SetStartNode(startID)
	if endID != "" {
		wf.SetEndNode(endID)
	}

	return wf, nil
}

// toNativeMap 转换字符串映射
func (w *WorkflowPluginNode) toNativeMap(sm models.StringMap) map[string]string {
	if len(sm) == 0 {
		return nil
	}
	out := make(map[string]string, len(sm))
	for k, v := range sm {
		out[k] = v
	}
	return out
}

// instantiateNode 实例化节点
func (w *WorkflowPluginNode) instantiateNode(base Node) (ExecutableNode, error) {
	switch base.Type {
	case NodeTypeStart:
		return &StartNode{Node: base}, nil
	case NodeTypeEnd:
		return &EndNode{Node: base}, nil
	case NodeTypeTask:
		taskNode := &TaskNode{Node: base}
		// 从属性中提取任务配置
		if base.Properties != nil {
			if taskType, ok := base.Properties["task_type"]; ok {
				taskNode.TaskType = taskType
			} else if taskType, ok := base.Properties["type"]; ok {
				taskNode.TaskType = taskType
			}
			if action, ok := base.Properties["action"]; ok {
				taskNode.Action = action
			}
			taskNode.Config = make(map[string]interface{})
			for k, v := range base.Properties {
				if k != "task_type" && k != "type" && k != "action" {
					taskNode.Config[k] = v
				}
			}
		}
		return taskNode, nil
	case NodeTypeScript:
		scriptNode := &ScriptNode{Node: base}
		if base.Properties != nil {
			if code, ok := base.Properties["code"]; ok {
				scriptNode.Script = code
			}
		}
		return scriptNode, nil
	case NodeTypeGateway:
		gatewayNode := &GatewayNode{Node: base}
		if base.Properties != nil {
			if condition, ok := base.Properties["condition"]; ok {
				gatewayNode.Condition = condition
			}
		}
		return gatewayNode, nil
	case NodeTypeAIChat:
		aiChatNode := &AIChatNode{Node: &base}
		if base.Properties != nil {
			// 将map[string]string转换为map[string]interface{}
			props := make(map[string]interface{})
			for k, v := range base.Properties {
				props[k] = v
			}
			config, err := ParseAIChatConfig(props)
			if err != nil {
				return nil, fmt.Errorf("parse AI chat config failed: %w", err)
			}
			aiChatNode.Config = config
		}
		return aiChatNode, nil
	default:
		// 对于其他类型的节点，创建一个基础的任务节点
		return &TaskNode{Node: base}, nil
	}
}
