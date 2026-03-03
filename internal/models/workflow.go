package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// WorkflowDefinition describes a reusable workflow template whose structure is stored as JSON graph data.
type WorkflowDefinition struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"userId" gorm:"index"`            // User ID
	GroupID          *uint          `json:"groupId,omitempty" gorm:"index"` // Organization ID, if set indicates this is an organization-shared workflow
	Name             string         `json:"name" gorm:"size:128;not null"`
	Slug             string         `json:"slug" gorm:"size:128;uniqueIndex"`
	Description      string         `json:"description" gorm:"type:text"`
	Version          uint           `json:"version" gorm:"default:1"`
	Status           string         `json:"status" gorm:"size:32;default:'draft'"` // draft, active, archived
	Definition       WorkflowGraph  `json:"definition" gorm:"type:json"`           // Node and connection JSON orchestration
	Settings         JSONMap        `json:"settings" gorm:"type:json"`             // Global configuration, such as default timeout, retry strategy
	Triggers         JSONMap        `json:"triggers,omitempty" gorm:"type:json"`   // Trigger configuration
	InputParameters  JSONArray      `json:"inputParameters" gorm:"type:json"`      // Input parameter definitions
	OutputParameters JSONArray      `json:"outputParameters" gorm:"type:json"`     // Output parameter definitions
	Tags             StringArray    `json:"tags" gorm:"type:json"`
	CreatedBy        string         `json:"createdBy" gorm:"size:64"`
	UpdatedBy        string         `json:"updatedBy" gorm:"size:64"`
	CreatedAt        time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// WorkflowInstance represents a runtime execution that references a workflow definition.
type WorkflowInstance struct {
	ID             uint                `json:"id" gorm:"primaryKey"`
	DefinitionID   uint                `json:"definitionId" gorm:"index"`
	DefinitionName string              `json:"definitionName" gorm:"size:128"`
	Status         string              `json:"status" gorm:"size:32;default:'pending'"` // pending,running,completed,failed
	CurrentNodeID  string              `json:"currentNodeId" gorm:"size:128"`
	ContextData    JSONMap             `json:"contextData" gorm:"type:json"` // Runtime context snapshot
	ResultData     JSONMap             `json:"resultData" gorm:"type:json"`
	StartedAt      *time.Time          `json:"startedAt"`
	CompletedAt    *time.Time          `json:"completedAt"`
	DeletedAt      gorm.DeletedAt      `json:"-" gorm:"index"`
	CreatedAt      time.Time           `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt      time.Time           `json:"updatedAt" gorm:"autoUpdateTime"`
	Definition     *WorkflowDefinition `json:"definition" gorm:"foreignKey:DefinitionID"`
}

// WorkflowGraph captures nodes and edges for a workflow definition serialized as JSON.
type WorkflowGraph struct {
	Nodes    []WorkflowNodeSchema `json:"nodes"`
	Edges    []WorkflowEdgeSchema `json:"edges"`
	Metadata JSONMap              `json:"metadata,omitempty"`
}

// WorkflowNodeSchema represents a single node within the workflow graph.
type WorkflowNodeSchema struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description,omitempty"`
	InputMap    StringMap `json:"inputMap,omitempty"`  // Input mapping, e.g. {"alias":"context.key"}
	OutputMap   StringMap `json:"outputMap,omitempty"` // Output mapping
	Properties  StringMap `json:"properties,omitempty"`
	Lanes       []string  `json:"lanes,omitempty"` // swimlane or grouping info
	Position    *Point    `json:"position,omitempty"`
}

// WorkflowEdgeSchema links nodes together and stores condition metadata.
type WorkflowEdgeSchema struct {
	ID          string           `json:"id"`
	Source      string           `json:"source"`
	Target      string           `json:"target"`
	Type        WorkflowEdgeType `json:"type,omitempty"`      // default,true,false,error,branch
	Condition   string           `json:"condition,omitempty"` // Expression or context key
	Description string           `json:"description,omitempty"`
	Metadata    JSONMap          `json:"metadata,omitempty"`
}

// WorkflowEdgeType enumerates semantic meaning of an edge.
type WorkflowEdgeType string

const (
	WorkflowEdgeTypeDefault WorkflowEdgeType = "default"
	WorkflowEdgeTypeTrue    WorkflowEdgeType = "true"
	WorkflowEdgeTypeFalse   WorkflowEdgeType = "false"
	WorkflowEdgeTypeError   WorkflowEdgeType = "error"
	WorkflowEdgeTypeBranch  WorkflowEdgeType = "branch"
)

// Point is used to store optional node coordinates for visual builders.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// WorkflowParameter represents a parameter definition for workflow input/output
type WorkflowParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, object, array
	Required    bool        `json:"required"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// JSONArray stores arrays as JSON.
type JSONArray []interface{}

// Value implements driver.Valuer.
func (ja JSONArray) Value() (driver.Value, error) {
	if len(ja) == 0 {
		return nil, nil
	}
	return json.Marshal(ja)
}

// Scan implements sql.Scanner.
func (ja *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*ja = []interface{}{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("JSONArray: expected []byte, got %T", value)
	}
	if len(bytes) == 0 {
		*ja = []interface{}{}
		return nil
	}
	return json.Unmarshal(bytes, ja)
}

// JSONMap allows flexible JSON storage for context/settings.
type JSONMap map[string]interface{}

// Value implements driver.Valuer.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil || len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner.
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(JSONMap)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("JSONMap: expected []byte, got %T", value)
	}
	if len(bytes) == 0 {
		*m = make(JSONMap)
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// Value implements driver.Valuer for WorkflowGraph.
func (g WorkflowGraph) Value() (driver.Value, error) {
	if len(g.Nodes) == 0 && len(g.Edges) == 0 && len(g.Metadata) == 0 {
		return nil, nil
	}
	return json.Marshal(g)
}

// Scan implements sql.Scanner for WorkflowGraph.
func (g *WorkflowGraph) Scan(value interface{}) error {
	if value == nil {
		*g = WorkflowGraph{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("WorkflowGraph: expected []byte, got %T", value)
	}
	if len(bytes) == 0 {
		*g = WorkflowGraph{}
		return nil
	}
	return json.Unmarshal(bytes, g)
}

// StringArray stores string slices as JSON.
type StringArray []string

// Value implements driver.Valuer.
func (sa StringArray) Value() (driver.Value, error) {
	if len(sa) == 0 {
		return nil, nil
	}
	return json.Marshal(sa)
}

// Scan implements sql.Scanner.
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("StringArray: expected []byte, got %T", value)
	}
	if len(bytes) == 0 {
		*sa = []string{}
		return nil
	}
	return json.Unmarshal(bytes, sa)
}

// StringMap stores string key/value pairs as JSON.
type StringMap map[string]string

// Value implements driver.Valuer.
func (sm StringMap) Value() (driver.Value, error) {
	if len(sm) == 0 {
		return nil, nil
	}
	return json.Marshal(sm)
}

// Scan implements sql.Scanner.
func (sm *StringMap) Scan(value interface{}) error {
	if value == nil {
		*sm = map[string]string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("StringMap: expected []byte, got %T", value)
	}
	if len(bytes) == 0 {
		*sm = map[string]string{}
		return nil
	}
	return json.Unmarshal(bytes, sm)
}

// WorkflowVersion stores historical versions of workflow definitions.
type WorkflowVersion struct {
	ID               uint                `json:"id" gorm:"primaryKey"`
	DefinitionID     uint                `json:"definitionId" gorm:"index;not null"`
	Version          uint                `json:"version" gorm:"not null"`
	Name             string              `json:"name" gorm:"size:128;not null"`
	Slug             string              `json:"slug" gorm:"size:128"`
	Description      string              `json:"description" gorm:"type:text"`
	Status           string              `json:"status" gorm:"size:32"`
	Definition       WorkflowGraph       `json:"definition" gorm:"type:json"`
	Settings         JSONMap             `json:"settings" gorm:"type:json"`
	Triggers         JSONMap             `json:"triggers,omitempty" gorm:"type:json"`
	InputParameters  JSONArray           `json:"inputParameters" gorm:"type:json"`  // Input parameter definitions
	OutputParameters JSONArray           `json:"outputParameters" gorm:"type:json"` // Output parameter definitions
	Tags             StringArray         `json:"tags" gorm:"type:json"`
	CreatedBy        string              `json:"createdBy" gorm:"size:64"`
	UpdatedBy        string              `json:"updatedBy" gorm:"size:64"`
	ChangeNote       string              `json:"changeNote" gorm:"type:text"` // Version change description
	CreatedAt        time.Time           `json:"createdAt" gorm:"autoCreateTime"`
	DefinitionRef    *WorkflowDefinition `json:"-" gorm:"foreignKey:DefinitionID"`
}

// MigrateWorkflowTables runs auto-migrations for workflow models.
func MigrateWorkflowTables(db *gorm.DB) error {
	return db.AutoMigrate(&WorkflowDefinition{}, &WorkflowInstance{}, &WorkflowVersion{})
}
