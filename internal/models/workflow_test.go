package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWorkflowTestDB(t *testing.T) *gorm.DB {
	return setupTestDBWithSilentLogger(t,
		&WorkflowDefinition{},
		&WorkflowInstance{},
		&WorkflowVersion{},
	)
}

func TestWorkflowDefinition_TableName(t *testing.T) {
	// WorkflowDefinition uses default table name from GORM
	// Verify it can be created
	db := setupWorkflowTestDB(t)
	def := &WorkflowDefinition{
		Name:        "Test Workflow",
		Slug:        "test-workflow",
		Description: "Test Description",
		Status:      "draft",
		UserID:      1,
	}
	err := db.Create(def).Error
	require.NoError(t, err)
	assert.NotZero(t, def.ID)
}

func TestWorkflowInstance_TableName(t *testing.T) {
	// WorkflowInstance uses default table name from GORM
	db := setupWorkflowTestDB(t)
	instance := &WorkflowInstance{
		DefinitionID:   1,
		DefinitionName: "Test Workflow",
		Status:         "pending",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)
	assert.NotZero(t, instance.ID)
}

func TestWorkflowVersion_TableName(t *testing.T) {
	// WorkflowVersion uses default table name from GORM
	db := setupWorkflowTestDB(t)
	version := &WorkflowVersion{
		DefinitionID: 1,
		Version:      1,
		Name:         "v1.0",
		Slug:         "v1-0",
		Status:       "draft",
	}
	err := db.Create(version).Error
	require.NoError(t, err)
	assert.NotZero(t, version.ID)
}

func TestMigrateWorkflowTables(t *testing.T) {
	db := setupWorkflowTestDB(t)

	// Migrate tables
	err := MigrateWorkflowTables(db)
	require.NoError(t, err)

	// Verify tables exist by creating records
	def := &WorkflowDefinition{
		Name:   "Test",
		Slug:   "test",
		UserID: 1,
	}
	err = db.Create(def).Error
	require.NoError(t, err)

	instance := &WorkflowInstance{
		DefinitionID:   def.ID,
		DefinitionName: "Test",
	}
	err = db.Create(instance).Error
	require.NoError(t, err)

	version := &WorkflowVersion{
		DefinitionID: def.ID,
		Version:      1,
		Name:         "v1",
	}
	err = db.Create(version).Error
	require.NoError(t, err)
}

// Test JSONMap Value and Scan
func TestJSONMap_Value(t *testing.T) {
	// Test with valid map
	m := JSONMap{
		"key1": "value1",
		"key2": 123,
		"key3": []string{"a", "b"},
	}
	value, err := m.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Test with nil map
	var m2 JSONMap
	value2, err := m2.Value()
	require.NoError(t, err)
	assert.Nil(t, value2)

	// Test with empty map
	m3 := JSONMap{}
	value3, err := m3.Value()
	require.NoError(t, err)
	assert.Nil(t, value3)
}

func TestJSONMap_Scan(t *testing.T) {
	// Test with valid bytes
	var m JSONMap
	bytes := []byte(`{"key":"value","number":123}`)
	err := m.Scan(bytes)
	require.NoError(t, err)
	assert.Equal(t, "value", m["key"])
	assert.Equal(t, float64(123), m["number"]) // JSON numbers become float64

	// Test with nil
	var m2 JSONMap
	err = m2.Scan(nil)
	require.NoError(t, err)
	assert.NotNil(t, m2)
	assert.Equal(t, 0, len(m2))

	// Test with empty bytes
	var m3 JSONMap
	err = m3.Scan([]byte{})
	require.NoError(t, err)
	assert.NotNil(t, m3)
	assert.Equal(t, 0, len(m3))

	// Test with invalid type
	var m4 JSONMap
	err = m4.Scan("not bytes")
	assert.Error(t, err)
}

// Test WorkflowGraph Value and Scan
func TestWorkflowGraph_Value(t *testing.T) {
	// Test with valid graph
	graph := WorkflowGraph{
		Nodes: []WorkflowNodeSchema{
			{
				ID:   "node1",
				Name: "Start",
				Type: "start",
			},
		},
		Edges: []WorkflowEdgeSchema{
			{
				ID:     "edge1",
				Source: "node1",
				Target: "node2",
			},
		},
		Metadata: JSONMap{
			"version": "1.0",
		},
	}
	value, err := graph.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Test with empty graph
	graph2 := WorkflowGraph{}
	value2, err := graph2.Value()
	require.NoError(t, err)
	assert.Nil(t, value2)
}

func TestWorkflowGraph_Scan(t *testing.T) {
	// Test with valid bytes
	var graph WorkflowGraph
	bytes := []byte(`{"nodes":[{"id":"node1","name":"Start","type":"start"}],"edges":[]}`)
	err := graph.Scan(bytes)
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 1)
	assert.Equal(t, "node1", graph.Nodes[0].ID)

	// Test with nil
	var graph2 WorkflowGraph
	err = graph2.Scan(nil)
	require.NoError(t, err)
	assert.Equal(t, WorkflowGraph{}, graph2)

	// Test with empty bytes
	var graph3 WorkflowGraph
	err = graph3.Scan([]byte{})
	require.NoError(t, err)
	assert.Equal(t, WorkflowGraph{}, graph3)

	// Test with invalid type
	var graph4 WorkflowGraph
	err = graph4.Scan("not bytes")
	assert.Error(t, err)
}

// Test StringArray Value and Scan
func TestStringArray_Value(t *testing.T) {
	// Test with valid array
	sa := StringArray{"tag1", "tag2", "tag3"}
	value, err := sa.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Test with empty array
	sa2 := StringArray{}
	value2, err := sa2.Value()
	require.NoError(t, err)
	assert.Nil(t, value2)
}

func TestStringArray_Scan(t *testing.T) {
	// Test with valid bytes
	var sa StringArray
	bytes := []byte(`["tag1","tag2","tag3"]`)
	err := sa.Scan(bytes)
	require.NoError(t, err)
	assert.Len(t, sa, 3)
	assert.Equal(t, "tag1", sa[0])

	// Test with nil
	var sa2 StringArray
	err = sa2.Scan(nil)
	require.NoError(t, err)
	assert.Len(t, sa2, 0)

	// Test with empty bytes
	var sa3 StringArray
	err = sa3.Scan([]byte{})
	require.NoError(t, err)
	assert.Len(t, sa3, 0)

	// Test with invalid type
	var sa4 StringArray
	err = sa4.Scan("not bytes")
	assert.Error(t, err)
}

// Test StringMap Value and Scan
func TestStringMap_Value(t *testing.T) {
	// Test with valid map
	sm := StringMap{
		"key1": "value1",
		"key2": "value2",
	}
	value, err := sm.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Test with empty map
	sm2 := StringMap{}
	value2, err := sm2.Value()
	require.NoError(t, err)
	assert.Nil(t, value2)
}

func TestStringMap_Scan(t *testing.T) {
	// Test with valid bytes
	var sm StringMap
	bytes := []byte(`{"key1":"value1","key2":"value2"}`)
	err := sm.Scan(bytes)
	require.NoError(t, err)
	assert.Equal(t, "value1", sm["key1"])
	assert.Equal(t, "value2", sm["key2"])

	// Test with nil
	var sm2 StringMap
	err = sm2.Scan(nil)
	require.NoError(t, err)
	assert.Len(t, sm2, 0)

	// Test with empty bytes
	var sm3 StringMap
	err = sm3.Scan([]byte{})
	require.NoError(t, err)
	assert.Len(t, sm3, 0)

	// Test with invalid type
	var sm4 StringMap
	err = sm4.Scan("not bytes")
	assert.Error(t, err)
}

func TestWorkflowDefinition_WithGraph(t *testing.T) {
	db := setupWorkflowTestDB(t)

	graph := WorkflowGraph{
		Nodes: []WorkflowNodeSchema{
			{
				ID:   "node1",
				Name: "Start Node",
				Type: "start",
				Position: &Point{
					X: 100,
					Y: 200,
				},
			},
			{
				ID:   "node2",
				Name: "Process Node",
				Type: "process",
				InputMap: StringMap{
					"input": "context.data",
				},
				OutputMap: StringMap{
					"output": "result.data",
				},
			},
		},
		Edges: []WorkflowEdgeSchema{
			{
				ID:     "edge1",
				Source: "node1",
				Target: "node2",
				Type:   WorkflowEdgeTypeDefault,
			},
		},
		Metadata: JSONMap{
			"version": "1.0",
		},
	}

	def := &WorkflowDefinition{
		UserID:      1,
		Name:        "Test Workflow",
		Slug:        "test-workflow",
		Description: "Test Description",
		Version:     1,
		Status:      "draft",
		Definition:  graph,
		Settings: JSONMap{
			"timeout": 30,
		},
		Triggers: JSONMap{
			"webhook": true,
		},
		Tags:      StringArray{"test", "demo"},
		CreatedBy: "user1",
		UpdatedBy: "user1",
	}

	err := db.Create(def).Error
	require.NoError(t, err)
	assert.NotZero(t, def.ID)

	// Retrieve and verify
	var retrieved WorkflowDefinition
	err = db.First(&retrieved, def.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "Test Workflow", retrieved.Name)
	assert.Len(t, retrieved.Definition.Nodes, 2)
	assert.Len(t, retrieved.Definition.Edges, 1)
	assert.Equal(t, "test", retrieved.Tags[0])
}

func TestWorkflowInstance_WithContext(t *testing.T) {
	db := setupWorkflowTestDB(t)

	// Create definition first
	def := &WorkflowDefinition{
		UserID: 1,
		Name:   "Test Workflow",
		Slug:   "test-workflow",
		Status: "active",
	}
	err := db.Create(def).Error
	require.NoError(t, err)

	instance := &WorkflowInstance{
		DefinitionID:   def.ID,
		DefinitionName: def.Name,
		Status:         "running",
		CurrentNodeID:  "node1",
		ContextData: JSONMap{
			"input": "test data",
		},
		ResultData: JSONMap{},
	}

	err = db.Create(instance).Error
	require.NoError(t, err)
	assert.NotZero(t, instance.ID)

	// Retrieve and verify
	var retrieved WorkflowInstance
	err = db.Preload("Definition").First(&retrieved, instance.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "running", retrieved.Status)
	assert.Equal(t, "test data", retrieved.ContextData["input"])
}

func TestWorkflowVersion_Creation(t *testing.T) {
	db := setupWorkflowTestDB(t)

	// Create definition first
	def := &WorkflowDefinition{
		UserID: 1,
		Name:   "Test Workflow",
		Slug:   "test-workflow",
		Status: "active",
	}
	err := db.Create(def).Error
	require.NoError(t, err)

	version := &WorkflowVersion{
		DefinitionID: def.ID,
		Version:      1,
		Name:         "v1.0",
		Slug:         "v1-0",
		Description:  "Initial version",
		Status:       "published",
		Definition: WorkflowGraph{
			Nodes: []WorkflowNodeSchema{
				{ID: "node1", Name: "Start", Type: "start"},
			},
		},
		Settings: JSONMap{
			"timeout": 30,
		},
		Tags:       StringArray{"v1"},
		CreatedBy:  "user1",
		UpdatedBy:  "user1",
		ChangeNote: "Initial release",
	}

	err = db.Create(version).Error
	require.NoError(t, err)
	assert.NotZero(t, version.ID)

	// Retrieve and verify
	var retrieved WorkflowVersion
	err = db.Preload("DefinitionRef").First(&retrieved, version.ID).Error
	require.NoError(t, err)
	assert.Equal(t, uint(1), retrieved.Version)
	assert.Equal(t, "v1.0", retrieved.Name)
	assert.Equal(t, "Initial release", retrieved.ChangeNote)
}

func TestWorkflowEdgeTypes(t *testing.T) {
	// Test all edge types
	edgeTypes := []WorkflowEdgeType{
		WorkflowEdgeTypeDefault,
		WorkflowEdgeTypeTrue,
		WorkflowEdgeTypeFalse,
		WorkflowEdgeTypeError,
		WorkflowEdgeTypeBranch,
	}

	for _, edgeType := range edgeTypes {
		edge := WorkflowEdgeSchema{
			ID:     "edge1",
			Source: "node1",
			Target: "node2",
			Type:   edgeType,
		}
		assert.Equal(t, edgeType, edge.Type)
	}
}

func TestWorkflowNodeSchema_WithLanes(t *testing.T) {
	node := WorkflowNodeSchema{
		ID:    "node1",
		Name:  "Test Node",
		Type:  "process",
		Lanes: []string{"lane1", "lane2"},
		Properties: StringMap{
			"prop1": "value1",
		},
	}

	assert.Len(t, node.Lanes, 2)
	assert.Equal(t, "value1", node.Properties["prop1"])
}

func TestPoint_Coordinates(t *testing.T) {
	point := &Point{
		X: 100.5,
		Y: 200.75,
	}

	assert.Equal(t, 100.5, point.X)
	assert.Equal(t, 200.75, point.Y)
}

func TestWorkflowDefinition_SoftDelete(t *testing.T) {
	db := setupWorkflowTestDB(t)

	def := &WorkflowDefinition{
		UserID: 1,
		Name:   "Test Workflow",
		Slug:   "test-workflow",
		Status: "draft",
	}
	err := db.Create(def).Error
	require.NoError(t, err)

	// Soft delete
	err = db.Delete(def).Error
	require.NoError(t, err)

	// Verify it's soft deleted
	var retrieved WorkflowDefinition
	err = db.First(&retrieved, def.ID).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// But should exist with Unscoped
	err = db.Unscoped().First(&retrieved, def.ID).Error
	require.NoError(t, err)
	assert.NotNil(t, retrieved.DeletedAt)
}

func TestWorkflowInstance_SoftDelete(t *testing.T) {
	db := setupWorkflowTestDB(t)

	instance := &WorkflowInstance{
		DefinitionID:   1,
		DefinitionName: "Test",
		Status:         "pending",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Soft delete
	err = db.Delete(instance).Error
	require.NoError(t, err)

	// Verify it's soft deleted
	var retrieved WorkflowInstance
	err = db.First(&retrieved, instance.ID).Error
	assert.Error(t, err)
}

func TestJSONMap_ComplexNesting(t *testing.T) {
	m := JSONMap{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "value",
			},
			"array": []interface{}{1, 2, 3},
		},
	}

	value, err := m.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Scan it back
	var m2 JSONMap
	err = m2.Scan(value.([]byte))
	require.NoError(t, err)
	assert.Contains(t, m2, "level1")
}

func TestWorkflowGraph_EmptyComponents(t *testing.T) {
	// Test with empty nodes but with edges
	graph := WorkflowGraph{
		Nodes: []WorkflowNodeSchema{},
		Edges: []WorkflowEdgeSchema{
			{ID: "edge1", Source: "node1", Target: "node2"},
		},
	}
	value, err := graph.Value()
	require.NoError(t, err)
	assert.NotNil(t, value) // Should not be nil because edges exist

	// Test with empty everything
	graph2 := WorkflowGraph{}
	value2, err := graph2.Value()
	require.NoError(t, err)
	assert.Nil(t, value2) // Should be nil when all empty
}

func TestStringArray_EmptyAndNil(t *testing.T) {
	// Test empty array
	sa1 := StringArray{}
	value1, err := sa1.Value()
	require.NoError(t, err)
	assert.Nil(t, value1)

	// Test nil scan
	var sa2 StringArray
	err = sa2.Scan(nil)
	require.NoError(t, err)
	assert.Len(t, sa2, 0)
}

func TestStringMap_EmptyAndNil(t *testing.T) {
	// Test empty map
	sm1 := StringMap{}
	value1, err := sm1.Value()
	require.NoError(t, err)
	assert.Nil(t, value1)

	// Test nil scan
	var sm2 StringMap
	err = sm2.Scan(nil)
	require.NoError(t, err)
	assert.Len(t, sm2, 0)
}

func TestWorkflowDefinition_GroupID(t *testing.T) {
	db := setupWorkflowTestDB(t)

	groupID := uint(100)
	def := &WorkflowDefinition{
		UserID:  1,
		GroupID: &groupID,
		Name:    "Shared Workflow",
		Slug:    "shared-workflow",
		Status:  "active",
	}
	err := db.Create(def).Error
	require.NoError(t, err)
	assert.NotNil(t, def.GroupID)
	assert.Equal(t, uint(100), *def.GroupID)
}

func TestWorkflowEdgeSchema_WithCondition(t *testing.T) {
	edge := WorkflowEdgeSchema{
		ID:          "edge1",
		Source:      "node1",
		Target:      "node2",
		Type:        WorkflowEdgeTypeBranch,
		Condition:   "context.value > 10",
		Description: "Conditional edge",
		Metadata: JSONMap{
			"priority": 1,
		},
	}

	assert.Equal(t, "context.value > 10", edge.Condition)
	assert.Equal(t, "Conditional edge", edge.Description)
	// JSON numbers can be int or float64 depending on how they're stored
	assert.NotNil(t, edge.Metadata["priority"])
	assert.True(t, edge.Metadata["priority"] == 1 || edge.Metadata["priority"] == float64(1))
}
