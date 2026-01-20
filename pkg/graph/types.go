package graph

// ConversationSummary 对话总结
type ConversationSummary struct {
	AssistantID   int64       `json:"assistantId"`
	AssistantName string      `json:"assistantName"`
	UserID        uint        `json:"userId"`
	SessionID     string      `json:"sessionId"`
	Summary       string      `json:"summary"`   // 对话总结
	Topics        []string    `json:"topics"`    // 讨论的主题列表
	Intents       []string    `json:"intents"`   // 用户意图列表
	Turns         []Turn      `json:"turns"`     // 对话轮次
	Knowledge     []Knowledge `json:"knowledge"` // 提取的知识点
}

// Turn 对话轮次
type Turn struct {
	UserMessage  string `json:"userMessage"`
	AgentMessage string `json:"agentMessage"`
	Sequence     int    `json:"sequence"`
}

// Knowledge 知识点
type Knowledge struct {
	Content       string   `json:"content"`       // 知识内容
	Category      string   `json:"category"`      // 知识类别
	Source        string   `json:"source"`        // 来源（如 "conversation"）
	RelatedTopics []string `json:"relatedTopics"` // 相关主题
}

// UserContext 用户上下文
type UserContext struct {
	UserID      uint     `json:"userId"`
	AssistantID int64    `json:"assistantId"`
	Topics      []string `json:"topics"` // 用户偏好的主题
}

// GraphNode 图节点
type GraphNode struct {
	ID    string                 `json:"id"`    // 节点ID
	Label string                 `json:"label"` // 节点标签
	Type  string                 `json:"type"`  // 节点类型（Assistant, User, Conversation, Topic, Intent, Knowledge等）
	Props map[string]interface{} `json:"props"` // 节点属性
}

// GraphEdge 图边（关系）
type GraphEdge struct {
	ID     string                 `json:"id"`     // 边ID
	Source string                 `json:"source"` // 源节点ID
	Target string                 `json:"target"` // 目标节点ID
	Type   string                 `json:"type"`   // 关系类型（HAS_CONVERSATION, WITH_ASSISTANT, DISCUSSES等）
	Props  map[string]interface{} `json:"props"`  // 边属性
}

// AssistantGraphData 助手图数据
type AssistantGraphData struct {
	AssistantID int64       `json:"assistantId"`
	Nodes       []GraphNode `json:"nodes"`
	Edges       []GraphEdge `json:"edges"`
	Stats       GraphStats  `json:"stats"`
}

// GraphStats 图统计信息
type GraphStats struct {
	TotalNodes         int `json:"totalNodes"`
	TotalEdges         int `json:"totalEdges"`
	UsersCount         int `json:"usersCount"`
	ConversationsCount int `json:"conversationsCount"`
	TopicsCount        int `json:"topicsCount"`
	IntentsCount       int `json:"intentsCount"`
	KnowledgeCount     int `json:"knowledgeCount"`
}
