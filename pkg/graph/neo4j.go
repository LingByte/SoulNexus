package graph

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

var globalStore Store

// SetDefaultStore 设置全局默认的图存储实例
func SetDefaultStore(store Store) {
	globalStore = store
}

// GetDefaultStore 获取全局默认的图存储实例
func GetDefaultStore() Store {
	return globalStore
}

// Store 图数据库存储接口
type Store interface {
	// ProcessConversation 处理对话记录，提取知识并存储到图数据库
	ProcessConversation(ctx context.Context, assistantID int64, sessionID string, summary *ConversationSummary) error

	// GetUserContext 获取用户上下文（偏好、历史主题等）
	GetUserContext(ctx context.Context, userID uint, assistantID int64) (*UserContext, error)

	// GetAssistantGraphData 获取助手在图数据库中的完整图数据
	GetAssistantGraphData(ctx context.Context, assistantID int64) (*AssistantGraphData, error)

	// Close 关闭连接
	Close() error
}

// Neo4jStore Neo4j 图数据库实现
type Neo4jStore struct {
	driver neo4j.DriverWithContext
	db     string // 数据库名称
}

// NewNeo4jStore 创建 Neo4j 存储实例
func NewNeo4jStore(uri, username, password, database string) (*Neo4jStore, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	store := &Neo4jStore{
		driver: driver,
		db:     database,
	}

	logger.Info("Neo4j connection established", zap.String("uri", uri), zap.String("database", database))
	return store, nil
}

// ProcessConversation 处理对话记录，提取知识并存储到图数据库
func (s *Neo4jStore) ProcessConversation(ctx context.Context, assistantID int64, sessionID string, summary *ConversationSummary) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.db,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// 1. 创建或更新 Assistant 节点
		assistantQuery := `
			MERGE (a:Assistant {id: $assistantID})
			SET a.name = $assistantName,
			    a.updatedAt = datetime()
			RETURN a`
		_, err := tx.Run(ctx, assistantQuery, map[string]any{
			"assistantID":   assistantID,
			"assistantName": summary.AssistantName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create/update assistant: %w", err)
		}

		// 2. 创建或更新 User 节点
		userQuery := `
			MERGE (u:User {id: $userID})
			SET u.updatedAt = datetime()
			RETURN u`
		_, err = tx.Run(ctx, userQuery, map[string]any{
			"userID": summary.UserID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create/update user: %w", err)
		}

		// 3. 创建 Conversation 节点
		conversationQuery := `
			MATCH (a:Assistant {id: $assistantID})
			MATCH (u:User {id: $userID})
			MERGE (c:Conversation {sessionID: $sessionID})
			SET c.assistantID = $assistantID,
			    c.userID = $userID,
			    c.summary = $summary,
			    c.topics = $topics,
			    c.intents = $intents,
			    c.createdAt = datetime(),
			    c.updatedAt = datetime()
			CREATE (u)-[:HAS_CONVERSATION]->(c)
			CREATE (c)-[:WITH_ASSISTANT]->(a)
			RETURN c`
		_, err = tx.Run(ctx, conversationQuery, map[string]any{
			"sessionID":   sessionID,
			"assistantID": assistantID,
			"userID":      summary.UserID,
			"summary":     summary.Summary,
			"topics":      summary.Topics,
			"intents":     summary.Intents,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create conversation: %w", err)
		}

		// 4. 创建 Turn 节点（对话轮次）
		for i, turn := range summary.Turns {
			turnQuery := `
				MATCH (c:Conversation {sessionID: $sessionID})
				MERGE (t:Turn {id: $turnID})
				SET t.userMessage = $userMessage,
				    t.agentMessage = $agentMessage,
				    t.sequence = $sequence,
				    t.createdAt = datetime()
				CREATE (c)-[:HAS_TURN]->(t)
				RETURN t`
			turnID := fmt.Sprintf("%s_turn_%d", sessionID, i)
			_, err = tx.Run(ctx, turnQuery, map[string]any{
				"sessionID":    sessionID,
				"turnID":       turnID,
				"userMessage":  turn.UserMessage,
				"agentMessage": turn.AgentMessage,
				"sequence":     i,
			})
			if err != nil {
				logger.Warn("failed to create turn", zap.Error(err), zap.Int("sequence", i))
				continue
			}
		}

		// 5. 创建 Topic 节点并建立关系
		for _, topic := range summary.Topics {
			topicQuery := `
				MATCH (c:Conversation {sessionID: $sessionID})
				MERGE (t:Topic {name: $topicName})
				SET t.updatedAt = datetime()
				MERGE (c)-[:DISCUSSES]->(t)
				RETURN t`
			_, err = tx.Run(ctx, topicQuery, map[string]any{
				"sessionID": sessionID,
				"topicName": topic,
			})
			if err != nil {
				logger.Warn("failed to create topic", zap.Error(err), zap.String("topic", topic))
				continue
			}

			// 建立用户与主题的关系（偏好）
			userTopicQuery := `
				MATCH (u:User {id: $userID})
				MATCH (t:Topic {name: $topicName})
				MERGE (u)-[r:LIKES]->(t)
				ON CREATE SET r.weight = 1, r.firstSeen = datetime()
				ON MATCH SET r.weight = r.weight + 1, r.lastSeen = datetime()
				RETURN r`
			_, err = tx.Run(ctx, userTopicQuery, map[string]any{
				"userID":    summary.UserID,
				"topicName": topic,
			})
			if err != nil {
				logger.Warn("failed to create user-topic relationship", zap.Error(err))
			}
		}

		// 6. 创建 Intent 节点并建立关系
		for _, intent := range summary.Intents {
			intentQuery := `
				MATCH (c:Conversation {sessionID: $sessionID})
				MERGE (i:Intent {name: $intentName})
				SET i.updatedAt = datetime()
				MERGE (c)-[:HAS_INTENT]->(i)
				RETURN i`
			_, err = tx.Run(ctx, intentQuery, map[string]any{
				"sessionID":  sessionID,
				"intentName": intent,
			})
			if err != nil {
				logger.Warn("failed to create intent", zap.Error(err), zap.String("intent", intent))
				continue
			}
		}

		// 7. 创建 Knowledge 节点（从对话中提取的知识点）
		for _, knowledge := range summary.Knowledge {
			knowledgeQuery := `
				MATCH (a:Assistant {id: $assistantID})
				MERGE (k:Knowledge {id: $knowledgeID})
				SET k.content = $content,
				    k.category = $category,
				    k.source = $source,
				    k.createdAt = datetime(),
				    k.updatedAt = datetime()
				MERGE (a)-[:HAS_KNOWLEDGE]->(k)
				RETURN k`
			knowledgeID := fmt.Sprintf("kg_%d_%d", assistantID, time.Now().UnixNano())
			_, err = tx.Run(ctx, knowledgeQuery, map[string]any{
				"assistantID": assistantID,
				"knowledgeID": knowledgeID,
				"content":     knowledge.Content,
				"category":    knowledge.Category,
				"source":      knowledge.Source,
			})
			if err != nil {
				logger.Warn("failed to create knowledge", zap.Error(err))
				continue
			}

			// 如果知识有相关主题，建立关系
			if len(knowledge.RelatedTopics) > 0 {
				for _, topic := range knowledge.RelatedTopics {
					knowledgeTopicQuery := `
						MATCH (k:Knowledge {id: $knowledgeID})
						MATCH (t:Topic {name: $topicName})
						MERGE (k)-[:RELATED_TO]->(t)
						RETURN k`
					_, err = tx.Run(ctx, knowledgeTopicQuery, map[string]any{
						"knowledgeID": knowledgeID,
						"topicName":   topic,
					})
					if err != nil {
						logger.Warn("failed to link knowledge to topic", zap.Error(err))
					}
				}
			}
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("failed to process conversation: %w", err)
	}

	logger.Info("Conversation processed and stored in graph",
		zap.Int64("assistantID", assistantID),
		zap.String("sessionID", sessionID))
	return nil
}

// GetUserContext 获取用户上下文（偏好、历史主题等）
func (s *Neo4jStore) GetUserContext(ctx context.Context, userID uint, assistantID int64) (*UserContext, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.db,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (u:User {id: $userID})-[r:LIKES]->(t:Topic)
			WHERE r.weight > 0
			WITH t, r.weight as weight
			ORDER BY weight DESC
			LIMIT 20
			RETURN collect(t.name) as topics`
		result, err := tx.Run(ctx, query, map[string]any{
			"userID": userID,
		})
		if err != nil {
			return nil, err
		}

		record, err := result.Single(ctx)
		if err != nil {
			// 如果没有记录，返回空上下文
			return &UserContext{
				UserID:      userID,
				AssistantID: assistantID,
				Topics:      []string{},
			}, nil
		}

		topics, _ := record.Get("topics")
		topicsList := []string{}
		if topics != nil {
			if topicsSlice, ok := topics.([]interface{}); ok {
				for _, t := range topicsSlice {
					if topicStr, ok := t.(string); ok {
						topicsList = append(topicsList, topicStr)
					}
				}
			}
		}

		return &UserContext{
			UserID:      userID,
			AssistantID: assistantID,
			Topics:      topicsList,
		}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	return result.(*UserContext), nil
}

// GetAssistantGraphData 获取助手在图数据库中的完整图数据
func (s *Neo4jStore) GetAssistantGraphData(ctx context.Context, assistantID int64) (*AssistantGraphData, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.db,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return s.getDetailedGraphData(tx, ctx, assistantID)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get assistant graph data: %w", err)
	}

	return result.(*AssistantGraphData), nil
}

// getDetailedGraphData 获取详细的图数据（当简单查询返回空时使用）
func (s *Neo4jStore) getDetailedGraphData(tx neo4j.ManagedTransaction, ctx context.Context, assistantID int64) (*AssistantGraphData, error) {
	nodesMap := make(map[string]GraphNode)
	edges := []GraphEdge{}

	// 1. 获取助手节点
	assistantQuery := `
		MATCH (a:Assistant {id: $assistantID})
		RETURN a`
	result, err := tx.Run(ctx, assistantQuery, map[string]any{"assistantID": assistantID})
	if err == nil {
		if record, err := result.Single(ctx); err == nil {
			if aNode, ok := record.Get("a"); ok {
				if node, ok := aNode.(neo4j.Node); ok {
					nodeID := fmt.Sprintf("Assistant_%d", assistantID)
					name := ""
					if nameVal, ok := node.Props["name"].(string); ok {
						name = nameVal
					}
					nodesMap[nodeID] = GraphNode{
						ID:    nodeID,
						Label: name,
						Type:  "Assistant",
						Props: node.Props,
					}
				}
			}
		}
	}

	// 2. 获取对话节点及其关系
	conversationQuery := `
		MATCH (a:Assistant {id: $assistantID})<-[:WITH_ASSISTANT]-(c:Conversation)
		OPTIONAL MATCH (u:User)-[:HAS_CONVERSATION]->(c)
		RETURN c, c.sessionID as sessionID, collect(DISTINCT u.id) as userIds`
	result, err = tx.Run(ctx, conversationQuery, map[string]any{"assistantID": assistantID})
	if err == nil {
		edgeCounter := 0
		for result.Next(ctx) {
			record := result.Record()
			if cNode, ok := record.Get("c"); ok {
				if node, ok := cNode.(neo4j.Node); ok {
					sessionID := ""
					if sessionVal, ok := record.Get("sessionID"); ok {
						if s, ok := sessionVal.(string); ok {
							sessionID = s
						}
					}
					nodeID := fmt.Sprintf("Conversation_%s", sessionID)
					label := sessionID
					if len(label) > 20 {
						label = label[:20] + "..."
					}
					nodesMap[nodeID] = GraphNode{
						ID:    nodeID,
						Label: label,
						Type:  "Conversation",
						Props: node.Props,
					}
					// 添加对话-助手边
					assistantNodeID := fmt.Sprintf("Assistant_%d", assistantID)
					edges = append(edges, GraphEdge{
						ID:     fmt.Sprintf("edge_conv_assistant_%d", edgeCounter),
						Source: nodeID,
						Target: assistantNodeID,
						Type:   "WITH_ASSISTANT",
						Props:  map[string]interface{}{},
					})
					edgeCounter++
					// 添加用户-对话边
					if userIds, ok := record.Get("userIds"); ok {
						if ids, ok := userIds.([]interface{}); ok {
							for _, idVal := range ids {
								userID := int64(0)
								// 处理不同的数字类型
								if id, ok := idVal.(int64); ok {
									userID = id
								} else if id, ok := idVal.(int32); ok {
									userID = int64(id)
								} else if id, ok := idVal.(float64); ok {
									userID = int64(id)
								}
								if userID > 0 {
									userNodeID := fmt.Sprintf("User_%d", userID)
									edges = append(edges, GraphEdge{
										ID:     fmt.Sprintf("edge_user_conv_%d_%d", edgeCounter, userID),
										Source: userNodeID,
										Target: nodeID,
										Type:   "HAS_CONVERSATION",
										Props:  map[string]interface{}{},
									})
									edgeCounter++
								}
							}
						}
					}
				}
			}
		}
	}

	// 3. 获取用户节点
	userQuery := `
		MATCH (a:Assistant {id: $assistantID})<-[:WITH_ASSISTANT]-(c:Conversation)<-[:HAS_CONVERSATION]-(u:User)
		RETURN DISTINCT u`
	result, err = tx.Run(ctx, userQuery, map[string]any{"assistantID": assistantID})
	if err == nil {
		for result.Next(ctx) {
			record := result.Record()
			if uNode, ok := record.Get("u"); ok {
				if node, ok := uNode.(neo4j.Node); ok {
					userID := int64(0)
					// 处理不同的数字类型
					if idVal, ok := node.Props["id"].(int64); ok {
						userID = idVal
					} else if idVal, ok := node.Props["id"].(int32); ok {
						userID = int64(idVal)
					} else if idVal, ok := node.Props["id"].(float64); ok {
						userID = int64(idVal)
					}
					nodeID := fmt.Sprintf("User_%d", userID)
					nodesMap[nodeID] = GraphNode{
						ID:    nodeID,
						Label: fmt.Sprintf("User %d", userID),
						Type:  "User",
						Props: node.Props,
					}
				}
			}
		}
	}

	// 4. 获取主题节点及其关系
	topicQuery := `
		MATCH (a:Assistant {id: $assistantID})<-[:WITH_ASSISTANT]-(c:Conversation)-[r:DISCUSSES]->(t:Topic)
		RETURN DISTINCT t, c.sessionID as sessionID`
	result, err = tx.Run(ctx, topicQuery, map[string]any{"assistantID": assistantID})
	if err == nil {
		edgeCounter := len(edges)
		for result.Next(ctx) {
			record := result.Record()
			if tNode, ok := record.Get("t"); ok {
				if node, ok := tNode.(neo4j.Node); ok {
					topicName := ""
					if nameVal, ok := node.Props["name"].(string); ok {
						topicName = nameVal
					}
					nodeID := fmt.Sprintf("Topic_%s", topicName)
					nodesMap[nodeID] = GraphNode{
						ID:    nodeID,
						Label: topicName,
						Type:  "Topic",
						Props: node.Props,
					}
					// 添加对话-主题边
					if sessionID, ok := record.Get("sessionID"); ok {
						if s, ok := sessionID.(string); ok {
							convNodeID := fmt.Sprintf("Conversation_%s", s)
							edges = append(edges, GraphEdge{
								ID:     fmt.Sprintf("edge_conv_topic_%d", edgeCounter),
								Source: convNodeID,
								Target: nodeID,
								Type:   "DISCUSSES",
								Props:  map[string]interface{}{},
							})
							edgeCounter++
						}
					}
				}
			}
		}
	}

	// 5. 获取意图节点及其关系
	intentQuery := `
		MATCH (a:Assistant {id: $assistantID})<-[:WITH_ASSISTANT]-(c:Conversation)-[r:HAS_INTENT]->(i:Intent)
		RETURN DISTINCT i, c.sessionID as sessionID`
	result, err = tx.Run(ctx, intentQuery, map[string]any{"assistantID": assistantID})
	if err == nil {
		edgeCounter := len(edges)
		for result.Next(ctx) {
			record := result.Record()
			if iNode, ok := record.Get("i"); ok {
				if node, ok := iNode.(neo4j.Node); ok {
					intentName := ""
					if nameVal, ok := node.Props["name"].(string); ok {
						intentName = nameVal
					}
					nodeID := fmt.Sprintf("Intent_%s", intentName)
					nodesMap[nodeID] = GraphNode{
						ID:    nodeID,
						Label: intentName,
						Type:  "Intent",
						Props: node.Props,
					}
					// 添加对话-意图边
					if sessionID, ok := record.Get("sessionID"); ok {
						if s, ok := sessionID.(string); ok {
							convNodeID := fmt.Sprintf("Conversation_%s", s)
							edges = append(edges, GraphEdge{
								ID:     fmt.Sprintf("edge_conv_intent_%d", edgeCounter),
								Source: convNodeID,
								Target: nodeID,
								Type:   "HAS_INTENT",
								Props:  map[string]interface{}{},
							})
							edgeCounter++
						}
					}
				}
			}
		}
	}

	// 6. 获取知识节点
	knowledgeQuery := `
		MATCH (a:Assistant {id: $assistantID})-[:HAS_KNOWLEDGE]->(k:Knowledge)
		RETURN DISTINCT k`
	result, err = tx.Run(ctx, knowledgeQuery, map[string]any{"assistantID": assistantID})
	if err == nil {
		for result.Next(ctx) {
			record := result.Record()
			if kNode, ok := record.Get("k"); ok {
				if node, ok := kNode.(neo4j.Node); ok {
					knowledgeID := ""
					if idVal, ok := node.Props["id"].(string); ok {
						knowledgeID = idVal
					}
					nodeID := fmt.Sprintf("Knowledge_%s", knowledgeID)
					content := ""
					if contentVal, ok := node.Props["content"].(string); ok {
						content = contentVal[:min(30, len(contentVal))] + "..."
					}
					nodesMap[nodeID] = GraphNode{
						ID:    nodeID,
						Label: content,
						Type:  "Knowledge",
						Props: node.Props,
					}
					// 添加边
					assistantNodeID := fmt.Sprintf("Assistant_%d", assistantID)
					edges = append(edges, GraphEdge{
						ID:     fmt.Sprintf("edge_knowledge_%s", knowledgeID),
						Source: assistantNodeID,
						Target: nodeID,
						Type:   "HAS_KNOWLEDGE",
						Props:  map[string]interface{}{},
					})
				}
			}
		}
	}

	// 转换为切片
	nodes := make([]GraphNode, 0, len(nodesMap))
	for _, node := range nodesMap {
		nodes = append(nodes, node)
	}

	// 计算统计信息
	stats := s.calculateStats(nodes)
	stats.TotalEdges = len(edges)

	return &AssistantGraphData{
		AssistantID: assistantID,
		Nodes:       nodes,
		Edges:       edges,
		Stats:       stats,
	}, nil
}

// calculateStats 计算图统计信息
func (s *Neo4jStore) calculateStats(nodes []GraphNode) GraphStats {
	stats := GraphStats{
		TotalNodes: len(nodes),
	}

	for _, node := range nodes {
		switch node.Type {
		case "User":
			stats.UsersCount++
		case "Conversation":
			stats.ConversationsCount++
		case "Topic":
			stats.TopicsCount++
		case "Intent":
			stats.IntentsCount++
		case "Knowledge":
			stats.KnowledgeCount++
		}
	}

	return stats
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Close 关闭连接
func (s *Neo4jStore) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.driver.Close(ctx)
}
