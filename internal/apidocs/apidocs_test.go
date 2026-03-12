package apidocs

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test structs for testing parseDocField
type TestUser struct {
	ID        uint       `json:"id" gorm:"primary" comment:"用户ID"`
	Name      string     `json:"name" binding:"required" comment:"用户名"`
	Email     *string    `json:"email,omitempty" comment:"邮箱"`
	Age       int        `json:"age" comment:"年龄"`
	Score     float64    `json:"score" comment:"分数"`
	IsActive  bool       `json:"is_active" comment:"是否激活"`
	CreatedAt time.Time  `json:"created_at" comment:"创建时间"`
	UpdatedAt time.Time  `json:"updated_at" comment:"更新时间"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" comment:"删除时间"`
}

type TestProfile struct {
	UserID uint   `json:"user_id" comment:"用户ID"`
	Avatar string `json:"avatar" comment:"头像"`
	Bio    string `json:"bio" comment:"简介"`
}

type TestUserWithProfile struct {
	TestUser
	Profile *TestProfile `json:"profile,omitempty" comment:"用户资料"`
}

type TestUserArray struct {
	Users []TestUser `json:"users" comment:"用户列表"`
}

type TestComplexStruct struct {
	ID       uint                   `json:"id"`
	Data     map[string]interface{} `json:"data"`
	Tags     []string               `json:"tags"`
	Metadata json.RawMessage        `json:"metadata"`
}

type TestEmbeddedStruct struct {
	TestUser
	Extra string `json:"extra" comment:"额外信息"`
}

type TestRecursiveStruct struct {
	ID       uint                   `json:"id"`
	Name     string                 `json:"name"`
	Children []*TestRecursiveStruct `json:"children,omitempty"`
}

// Test WebObject implementation
type TestWebObject struct {
	ID   uint   `json:"id" gorm:"primary"`
	Name string `json:"name"`
}

func (TestWebObject) TableName() string {
	return "test_objects"
}

func TestGetDocDefine(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *DocField
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:  "simple struct",
			input: TestUser{},
			expected: &DocField{
				Name: "",
				Type: TYPE_OBJECT,
				Fields: []DocField{
					{
						FieldName: "ID",
						Name:      "id",
						Type:      TYPE_INT,
						Desc:      "用户ID",
						IsPrimary: true,
					},
					{
						FieldName: "Name",
						Name:      "name",
						Type:      TYPE_STRING,
						Desc:      "用户名",
						Required:  true,
					},
					{
						FieldName: "Email",
						Name:      "email",
						Type:      TYPE_STRING,
						Desc:      "邮箱",
						CanNull:   true,
					},
					{
						FieldName: "Age",
						Name:      "age",
						Type:      TYPE_INT,
						Desc:      "年龄",
					},
					{
						FieldName: "Score",
						Name:      "score",
						Type:      TYPE_FLOAT,
						Desc:      "分数",
					},
					{
						FieldName: "IsActive",
						Name:      "is_active",
						Type:      TYPE_BOOLEAN,
						Desc:      "是否激活",
					},
					{
						FieldName: "CreatedAt",
						Name:      "created_at",
						Type:      TYPE_DATE,
						Desc:      "创建时间",
					},
					{
						FieldName: "UpdatedAt",
						Name:      "updated_at",
						Type:      TYPE_DATE,
						Desc:      "更新时间",
					},
					{
						FieldName: "DeletedAt",
						Name:      "deleted_at",
						Type:      TYPE_DATE,
						Desc:      "删除时间",
						CanNull:   true,
					},
				},
			},
		},
		{
			name:  "struct with nested struct",
			input: TestUserWithProfile{},
			expected: &DocField{
				Name: "",
				Type: TYPE_OBJECT,
				Fields: []DocField{
					// TestUser fields (embedded)
					{
						FieldName: "ID",
						Name:      "id",
						Type:      TYPE_INT,
						Desc:      "用户ID",
						IsPrimary: true,
					},
					{
						FieldName: "Name",
						Name:      "name",
						Type:      TYPE_STRING,
						Desc:      "用户名",
						Required:  true,
					},
					{
						FieldName: "Email",
						Name:      "email",
						Type:      TYPE_STRING,
						Desc:      "邮箱",
						CanNull:   true,
					},
					{
						FieldName: "Age",
						Name:      "age",
						Type:      TYPE_INT,
						Desc:      "年龄",
					},
					{
						FieldName: "Score",
						Name:      "score",
						Type:      TYPE_FLOAT,
						Desc:      "分数",
					},
					{
						FieldName: "IsActive",
						Name:      "is_active",
						Type:      TYPE_BOOLEAN,
						Desc:      "是否激活",
					},
					{
						FieldName: "CreatedAt",
						Name:      "created_at",
						Type:      TYPE_DATE,
						Desc:      "创建时间",
					},
					{
						FieldName: "UpdatedAt",
						Name:      "updated_at",
						Type:      TYPE_DATE,
						Desc:      "更新时间",
					},
					{
						FieldName: "DeletedAt",
						Name:      "deleted_at",
						Type:      TYPE_DATE,
						Desc:      "删除时间",
						CanNull:   true,
					},
					// Profile field
					{
						FieldName: "Profile",
						Name:      "profile",
						Type:      TYPE_OBJECT,
						Desc:      "用户资料",
						CanNull:   true,
						Fields: []DocField{
							{
								FieldName: "UserID",
								Name:      "user_id",
								Type:      TYPE_INT,
								Desc:      "用户ID",
							},
							{
								FieldName: "Avatar",
								Name:      "avatar",
								Type:      TYPE_STRING,
								Desc:      "头像",
							},
							{
								FieldName: "Bio",
								Name:      "bio",
								Type:      TYPE_STRING,
								Desc:      "简介",
							},
						},
					},
				},
			},
		},
		{
			name:  "struct with array",
			input: TestUserArray{},
			expected: &DocField{
				Name: "",
				Type: TYPE_OBJECT,
				Fields: []DocField{
					{
						FieldName: "Users",
						Name:      "users",
						Type:      TYPE_OBJECT,
						Desc:      "用户列表",
						IsArray:   true,
						Fields: []DocField{
							{
								FieldName: "ID",
								Name:      "id",
								Type:      TYPE_INT,
								Desc:      "用户ID",
								IsPrimary: true,
							},
							{
								FieldName: "Name",
								Name:      "name",
								Type:      TYPE_STRING,
								Desc:      "用户名",
								Required:  true,
							},
							{
								FieldName: "Email",
								Name:      "email",
								Type:      TYPE_STRING,
								Desc:      "邮箱",
								CanNull:   true,
							},
							{
								FieldName: "Age",
								Name:      "age",
								Type:      TYPE_INT,
								Desc:      "年龄",
							},
							{
								FieldName: "Score",
								Name:      "score",
								Type:      TYPE_FLOAT,
								Desc:      "分数",
							},
							{
								FieldName: "IsActive",
								Name:      "is_active",
								Type:      TYPE_BOOLEAN,
								Desc:      "是否激活",
							},
							{
								FieldName: "CreatedAt",
								Name:      "created_at",
								Type:      TYPE_DATE,
								Desc:      "创建时间",
							},
							{
								FieldName: "UpdatedAt",
								Name:      "updated_at",
								Type:      TYPE_DATE,
								Desc:      "更新时间",
							},
							{
								FieldName: "DeletedAt",
								Name:      "deleted_at",
								Type:      TYPE_DATE,
								Desc:      "删除时间",
								CanNull:   true,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDocDefine(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, len(tt.expected.Fields), len(result.Fields))

			// Compare fields
			for i, expectedField := range tt.expected.Fields {
				if i < len(result.Fields) {
					actualField := result.Fields[i]
					assert.Equal(t, expectedField.FieldName, actualField.FieldName, "Field %d FieldName mismatch", i)
					assert.Equal(t, expectedField.Name, actualField.Name, "Field %d Name mismatch", i)
					assert.Equal(t, expectedField.Type, actualField.Type, "Field %d Type mismatch", i)
					assert.Equal(t, expectedField.Desc, actualField.Desc, "Field %d Desc mismatch", i)
					assert.Equal(t, expectedField.Required, actualField.Required, "Field %d Required mismatch", i)
					assert.Equal(t, expectedField.CanNull, actualField.CanNull, "Field %d CanNull mismatch", i)
					assert.Equal(t, expectedField.IsArray, actualField.IsArray, "Field %d IsArray mismatch", i)
					assert.Equal(t, expectedField.IsPrimary, actualField.IsPrimary, "Field %d IsPrimary mismatch", i)
				}
			}
		})
	}
}

func TestParseType(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		{
			name:     "string type",
			input:    reflect.TypeOf(""),
			expected: TYPE_STRING,
		},
		{
			name:     "int type",
			input:    reflect.TypeOf(0),
			expected: TYPE_INT,
		},
		{
			name:     "int64 type",
			input:    reflect.TypeOf(int64(0)),
			expected: TYPE_INT,
		},
		{
			name:     "uint type",
			input:    reflect.TypeOf(uint(0)),
			expected: TYPE_INT,
		},
		{
			name:     "float64 type",
			input:    reflect.TypeOf(0.0),
			expected: TYPE_FLOAT,
		},
		{
			name:     "bool type",
			input:    reflect.TypeOf(true),
			expected: TYPE_BOOLEAN,
		},
		{
			name:     "time.Time type",
			input:    reflect.TypeOf(time.Time{}),
			expected: TYPE_DATE,
		},
		{
			name:     "map type",
			input:    reflect.TypeOf(map[string]interface{}{}),
			expected: TYPE_MAP,
		},
		{
			name:     "slice type",
			input:    reflect.TypeOf([]string{}),
			expected: TYPE_STRING,
		},
		{
			name:     "array type",
			input:    reflect.TypeOf([5]int{}),
			expected: TYPE_INT,
		},
		{
			name:     "pointer type",
			input:    reflect.TypeOf((*string)(nil)),
			expected: TYPE_STRING,
		},
		{
			name:     "double pointer type",
			input:    reflect.TypeOf((**string)(nil)),
			expected: TYPE_OBJECT,
		},
		{
			name:     "struct slice type",
			input:    reflect.TypeOf([]TestUser{}),
			expected: TYPE_OBJECT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDocField(t *testing.T) {
	tests := []struct {
		name      string
		rt        reflect.Type
		fieldName string
		expected  DocField
	}{
		{
			name:      "string field",
			rt:        reflect.TypeOf(""),
			fieldName: "test",
			expected: DocField{
				Name: "test",
				Type: TYPE_STRING,
			},
		},
		{
			name:      "pointer field",
			rt:        reflect.TypeOf((*string)(nil)),
			fieldName: "test",
			expected: DocField{
				Name:    "test",
				Type:    TYPE_STRING,
				CanNull: true,
			},
		},
		{
			name:      "array field",
			rt:        reflect.TypeOf([]string{}),
			fieldName: "test",
			expected: DocField{
				Name:    "test",
				Type:    TYPE_STRING,
				IsArray: true,
			},
		},
		{
			name:      "struct field",
			rt:        reflect.TypeOf(TestUser{}),
			fieldName: "user",
			expected: DocField{
				Name: "user",
				Type: TYPE_OBJECT,
				Fields: []DocField{
					{
						FieldName: "ID",
						Name:      "id",
						Type:      TYPE_INT,
						Desc:      "用户ID",
						IsPrimary: true,
					},
					{
						FieldName: "Name",
						Name:      "name",
						Type:      TYPE_STRING,
						Desc:      "用户名",
						Required:  true,
					},
					{
						FieldName: "Email",
						Name:      "email",
						Type:      TYPE_STRING,
						Desc:      "邮箱",
						CanNull:   true,
					},
					{
						FieldName: "Age",
						Name:      "age",
						Type:      TYPE_INT,
						Desc:      "年龄",
					},
					{
						FieldName: "Score",
						Name:      "score",
						Type:      TYPE_FLOAT,
						Desc:      "分数",
					},
					{
						FieldName: "IsActive",
						Name:      "is_active",
						Type:      TYPE_BOOLEAN,
						Desc:      "是否激活",
					},
					{
						FieldName: "CreatedAt",
						Name:      "created_at",
						Type:      TYPE_DATE,
						Desc:      "创建时间",
					},
					{
						FieldName: "UpdatedAt",
						Name:      "updated_at",
						Type:      TYPE_DATE,
						Desc:      "更新时间",
					},
					{
						FieldName: "DeletedAt",
						Name:      "deleted_at",
						Type:      TYPE_DATE,
						Desc:      "删除时间",
						CanNull:   true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDocField(tt.rt, tt.fieldName, nil)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.CanNull, result.CanNull)
			assert.Equal(t, tt.expected.IsArray, result.IsArray)

			if len(tt.expected.Fields) > 0 {
				assert.Equal(t, len(tt.expected.Fields), len(result.Fields))
				for i, expectedField := range tt.expected.Fields {
					if i < len(result.Fields) {
						actualField := result.Fields[i]
						assert.Equal(t, expectedField.FieldName, actualField.FieldName)
						assert.Equal(t, expectedField.Name, actualField.Name)
						assert.Equal(t, expectedField.Type, actualField.Type)
						assert.Equal(t, expectedField.Desc, actualField.Desc)
						assert.Equal(t, expectedField.Required, actualField.Required)
						assert.Equal(t, expectedField.CanNull, actualField.CanNull)
						assert.Equal(t, expectedField.IsPrimary, actualField.IsPrimary)
					}
				}
			}
		})
	}
}

func TestGetWebObjectDocDefine(t *testing.T) {
	// Create a test WebObject
	webObj := LingEcho.WebObject{
		Name:         "users",
		Group:        "User Management",
		Desc:         "User management endpoints",
		AuthRequired: true,
		Model:        TestUser{},
		AllowMethods: LingEcho.GET | LingEcho.CREATE | LingEcho.EDIT | LingEcho.DELETE,
		Filterables:  []string{"name", "email"},
		Orderables:   []string{"id", "created_at"},
		Searchables:  []string{"name", "email"},
		Editables:    []string{"Name", "Email", "Age"},
		Views: []LingEcho.QueryView{
			{
				Path:   "/profile",
				Method: "GET",
				Desc:   "Get user profile",
			},
		},
	}

	result := GetWebObjectDocDefine("/api", webObj)

	assert.Equal(t, "User Management", result.Group)
	assert.Equal(t, "/api/users", result.Path)
	assert.Equal(t, "User management endpoints", result.Desc)
	assert.True(t, result.AuthRequired)
	assert.Contains(t, result.AllowMethods, "GET")
	assert.Contains(t, result.AllowMethods, "CREATE")
	assert.Contains(t, result.AllowMethods, "EDIT")
	assert.Contains(t, result.AllowMethods, "DELETE")
	assert.NotContains(t, result.AllowMethods, "QUERY")
	assert.Equal(t, []string{"name", "email"}, result.Filters)
	assert.Equal(t, []string{"id", "created_at"}, result.Orders)
	assert.Equal(t, []string{"name", "email"}, result.Searches)
	assert.Equal(t, []string{"name", "email", "age"}, result.Editables)
	assert.Len(t, result.Views, 1)
	assert.Equal(t, "/api/users/profile", result.Views[0].Path)
	assert.Equal(t, "GET", result.Views[0].Method)
	assert.Equal(t, "Get user profile", result.Views[0].Desc)

	// Test with default AllowMethods (0)
	webObj.AllowMethods = 0
	result = GetWebObjectDocDefine("/api", webObj)
	assert.Contains(t, result.AllowMethods, "GET")
	assert.Contains(t, result.AllowMethods, "CREATE")
	assert.Contains(t, result.AllowMethods, "EDIT")
	assert.Contains(t, result.AllowMethods, "DELETE")
	assert.Contains(t, result.AllowMethods, "QUERY")

	// Test with empty Editables (should use all fields)
	webObj.Editables = []string{}
	result = GetWebObjectDocDefine("/api", webObj)
	assert.Contains(t, result.Editables, "id")
	assert.Contains(t, result.Editables, "name")
	assert.Contains(t, result.Editables, "email")
}

func TestRegisterHandler(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Test data
	uriDocs := []UriDoc{
		{
			Group:   "Test",
			Path:    "/test",
			Summary: "Test endpoint",
			Desc:    "Test description",
			Method:  "GET",
			Request: &DocField{
				Type: TYPE_OBJECT,
				Fields: []DocField{
					{Name: "id", Type: TYPE_INT},
				},
			},
			Response: &DocField{
				Type: TYPE_OBJECT,
				Fields: []DocField{
					{Name: "message", Type: TYPE_STRING},
				},
			},
		},
	}

	objDocs := []WebObjectDoc{
		{
			Group: "Test",
			Path:  "/objects",
			Desc:  "Test objects",
		},
	}

	// Register handler
	RegisterHandler("/docs", router, uriDocs, objDocs, db)

	// Test JSON endpoint
	t.Run("JSON endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/docs.json", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "uris")
		assert.Contains(t, response, "objs")
		assert.Contains(t, response, "site")
	})

	// Test HTML endpoint
	t.Run("HTML endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/docs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
		assert.Contains(t, w.Body.String(), "API Docs")
	})

	// Test with trailing slash in prefix
	t.Run("Trailing slash handling", func(t *testing.T) {
		router2 := gin.New()
		RegisterHandler("/docs/", router2, uriDocs, objDocs, db)

		req := httptest.NewRequest("GET", "/docs.json", nil)
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestComplexStructParsing(t *testing.T) {
	t.Run("Complex struct with maps and raw message", func(t *testing.T) {
		result := GetDocDefine(TestComplexStruct{})
		assert.NotNil(t, result)
		assert.Equal(t, TYPE_OBJECT, result.Type)
		assert.Len(t, result.Fields, 4)

		// Check ID field
		idField := result.Fields[0]
		assert.Equal(t, "id", idField.Name)
		assert.Equal(t, TYPE_INT, idField.Type)

		// Check Data field (map)
		dataField := result.Fields[1]
		assert.Equal(t, "data", dataField.Name)
		assert.Equal(t, TYPE_MAP, dataField.Type)

		// Check Tags field (array)
		tagsField := result.Fields[2]
		assert.Equal(t, "tags", tagsField.Name)
		assert.Equal(t, TYPE_STRING, tagsField.Type)
		assert.True(t, tagsField.IsArray)

		// Check Metadata field
		metadataField := result.Fields[3]
		assert.Equal(t, "metadata", metadataField.Name)
	})
}

func TestEmbeddedStructParsing(t *testing.T) {
	t.Run("Embedded struct", func(t *testing.T) {
		result := GetDocDefine(TestEmbeddedStruct{})
		assert.NotNil(t, result)
		assert.Equal(t, TYPE_OBJECT, result.Type)

		// Should have all TestUser fields plus Extra field
		expectedFieldCount := 10 // 9 from TestUser + 1 Extra
		assert.Len(t, result.Fields, expectedFieldCount)

		// Check that embedded fields are included
		var foundID, foundName, foundExtra bool
		for _, field := range result.Fields {
			switch field.Name {
			case "id":
				foundID = true
				assert.Equal(t, TYPE_INT, field.Type)
				assert.True(t, field.IsPrimary)
			case "name":
				foundName = true
				assert.Equal(t, TYPE_STRING, field.Type)
				assert.True(t, field.Required)
			case "extra":
				foundExtra = true
				assert.Equal(t, TYPE_STRING, field.Type)
				assert.Equal(t, "额外信息", field.Desc)
			}
		}
		assert.True(t, foundID, "ID field should be present")
		assert.True(t, foundName, "Name field should be present")
		assert.True(t, foundExtra, "Extra field should be present")
	})
}

func TestRecursiveStructParsing(t *testing.T) {
	t.Run("Recursive struct", func(t *testing.T) {
		result := GetDocDefine(TestRecursiveStruct{})
		assert.NotNil(t, result)
		assert.Equal(t, TYPE_OBJECT, result.Type)
		assert.Len(t, result.Fields, 3)

		// Check Children field
		var childrenField *DocField
		for _, field := range result.Fields {
			if field.Name == "children" {
				childrenField = &field
				break
			}
		}
		assert.NotNil(t, childrenField)
		assert.Equal(t, TYPE_OBJECT, childrenField.Type)
		assert.True(t, childrenField.IsArray)
		assert.True(t, childrenField.CanNull)
		// Recursive fields should be empty to prevent infinite recursion
		assert.Empty(t, childrenField.Fields)
	})
}

func TestJSONTagParsing(t *testing.T) {
	type TestJSONTags struct {
		Field1 string `json:"custom_name"`
		Field2 string `json:"field2,omitempty"`
		Field3 string `json:"-"`
		Field4 string // no json tag
	}

	result := GetDocDefine(TestJSONTags{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 3) // Field3 should be excluded

	fieldNames := make(map[string]bool)
	for _, field := range result.Fields {
		fieldNames[field.Name] = true
		if field.Name == "custom_name" {
			assert.Equal(t, "Field1", field.FieldName)
		}
		if field.Name == "field2" {
			assert.True(t, field.CanNull) // omitempty should set CanNull
		}
	}

	assert.True(t, fieldNames["custom_name"])
	assert.True(t, fieldNames["field2"])
	assert.True(t, fieldNames["Field4"])  // no json tag, use field name
	assert.False(t, fieldNames["field3"]) // should be excluded
}

func TestGormTagParsing(t *testing.T) {
	type TestGormTags struct {
		ID   uint   `gorm:"primary"`
		Name string `gorm:"not null"`
	}

	result := GetDocDefine(TestGormTags{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 2)

	for _, field := range result.Fields {
		if field.Name == "ID" {
			assert.True(t, field.IsPrimary)
		}
	}
}

func TestBindingTagParsing(t *testing.T) {
	type TestBindingTags struct {
		Required string `binding:"required"`
		Optional string
	}

	result := GetDocDefine(TestBindingTags{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 2)

	for _, field := range result.Fields {
		if field.Name == "Required" {
			assert.True(t, field.Required)
		} else if field.Name == "Optional" {
			assert.False(t, field.Required)
		}
	}
}

func TestNullTypes(t *testing.T) {
	// Test various null types that might be used
	type TestNullTypes struct {
		NullString *string
		NullInt    *int
		NullBool   *bool
		NullFloat  *float64
		NullTime   *time.Time
	}

	result := GetDocDefine(TestNullTypes{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 5)

	for _, field := range result.Fields {
		assert.True(t, field.CanNull, "Field %s should be nullable", field.Name)
	}
}

// Test constants
func TestConstants(t *testing.T) {
	assert.Equal(t, "date", TYPE_DATE)
	assert.Equal(t, "string", TYPE_STRING)
	assert.Equal(t, "int", TYPE_INT)
	assert.Equal(t, "float", TYPE_FLOAT)
	assert.Equal(t, "boolean", TYPE_BOOLEAN)
	assert.Equal(t, "object", TYPE_OBJECT)
	assert.Equal(t, "map", TYPE_MAP)
}

// Benchmark tests
func BenchmarkGetDocDefine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetDocDefine(TestUser{})
	}
}

func BenchmarkParseDocField(b *testing.B) {
	rt := reflect.TypeOf(TestUser{})
	for i := 0; i < b.N; i++ {
		parseDocField(rt, "test", nil)
	}
}

func BenchmarkParseType(b *testing.B) {
	rt := reflect.TypeOf("")
	for i := 0; i < b.N; i++ {
		parseType(rt)
	}
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("Empty struct", func(t *testing.T) {
		type EmptyStruct struct{}
		result := GetDocDefine(EmptyStruct{})
		assert.NotNil(t, result)
		assert.Equal(t, TYPE_OBJECT, result.Type)
		assert.Empty(t, result.Fields)
	})

	t.Run("Struct with only ignored fields", func(t *testing.T) {
		type IgnoredStruct struct {
			Field1 string `json:"-"`
			Field2 string `json:"-"`
		}
		result := GetDocDefine(IgnoredStruct{})
		assert.NotNil(t, result)
		assert.Equal(t, TYPE_OBJECT, result.Type)
		assert.Empty(t, result.Fields)
	})

	t.Run("Primitive types", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected string
		}{
			{"string", TYPE_STRING},
			{123, TYPE_INT},
			{123.45, TYPE_FLOAT},
			{true, TYPE_BOOLEAN},
			{time.Now(), TYPE_DATE},
		}

		for _, test := range tests {
			result := GetDocDefine(test.input)
			assert.NotNil(t, result)
			assert.Equal(t, test.expected, result.Type)
		}
	})
}

// Test with interface{} and any types
func TestInterfaceTypes(t *testing.T) {
	type TestInterface struct {
		Data interface{} `json:"data"`
		Any  any         `json:"any"`
	}

	result := GetDocDefine(TestInterface{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 2)

	for _, field := range result.Fields {
		// interface{} and any should be parsed as empty type
		assert.Equal(t, "", field.Type)
	}
}

// Test SQL null types
func TestSQLNullTypes(t *testing.T) {
	// Create custom types that match the names checked in parseDocField
	type NullTime struct{}
	type NullBool struct{}
	type NullString struct{}
	type NullByte struct{}
	type NullInt16 struct{}
	type NullInt32 struct{}
	type NullInt64 struct{}
	type NullFloat32 struct{}
	type NullFloat64 struct{}
	type DeletedAt struct{}

	type TestSQLNulls struct {
		NullTime    NullTime    `json:"null_time"`
		NullBool    NullBool    `json:"null_bool"`
		NullString  NullString  `json:"null_string"`
		NullByte    NullByte    `json:"null_byte"`
		NullInt16   NullInt16   `json:"null_int16"`
		NullInt32   NullInt32   `json:"null_int32"`
		NullInt64   NullInt64   `json:"null_int64"`
		NullFloat32 NullFloat32 `json:"null_float32"`
		NullFloat64 NullFloat64 `json:"null_float64"`
		DeletedAt   DeletedAt   `json:"deleted_at"`
	}

	result := GetDocDefine(TestSQLNulls{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 10)

	// Check each field individually
	for _, field := range result.Fields {
		// Fields starting with "Null" should be marked as nullable
		if strings.HasPrefix(field.FieldName, "Null") {
			assert.True(t, field.CanNull, "Field %s should be nullable", field.Name)
		}

		// Check specific type mappings based on the switch statement in parseDocField
		switch field.FieldName {
		case "NullTime", "NullBool", "NullString", "NullByte", "NullInt16",
			"NullInt32", "NullInt64", "NullFloat32", "NullFloat64":
			// These are handled by the first case in the switch, so they get object type
			assert.Equal(t, TYPE_OBJECT, field.Type)
		case "DeletedAt":
			// This is handled by the second case and returns early with date type
			assert.Equal(t, TYPE_DATE, field.Type)
		}
	}
}

// Test Time type specifically
func TestTimeTypes(t *testing.T) {
	type TestTimeTypes struct {
		Time      time.Time `json:"time"`
		DeletedAt time.Time `json:"deleted_at"`
	}

	result := GetDocDefine(TestTimeTypes{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 2)

	for _, field := range result.Fields {
		assert.Equal(t, TYPE_DATE, field.Type)
	}
}

// Test complex nested scenarios
func TestComplexNestedScenarios(t *testing.T) {
	type Level3 struct {
		Value string `json:"value"`
	}

	type Level2 struct {
		Level3 Level3 `json:"level3"`
	}

	type Level1 struct {
		Level2 Level2 `json:"level2"`
	}

	result := GetDocDefine(Level1{})
	assert.NotNil(t, result)
	assert.Equal(t, TYPE_OBJECT, result.Type)
	assert.Len(t, result.Fields, 1)

	level2Field := result.Fields[0]
	assert.Equal(t, "level2", level2Field.Name)
	assert.Equal(t, TYPE_OBJECT, level2Field.Type)
	assert.Len(t, level2Field.Fields, 1)

	level3Field := level2Field.Fields[0]
	assert.Equal(t, "level3", level3Field.Name)
	assert.Equal(t, TYPE_OBJECT, level3Field.Type)
	assert.Len(t, level3Field.Fields, 1)

	valueField := level3Field.Fields[0]
	assert.Equal(t, "value", valueField.Name)
	assert.Equal(t, TYPE_STRING, valueField.Type)
}

// Test anonymous struct fields
func TestAnonymousStructFields(t *testing.T) {
	type BaseStruct struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}

	type ExtendedStruct struct {
		BaseStruct        // Anonymous field
		Extra      string `json:"extra"`
	}

	result := GetDocDefine(ExtendedStruct{})
	assert.NotNil(t, result)
	assert.Equal(t, TYPE_OBJECT, result.Type)

	// Should have 3 fields: ID, Name (from embedded), and Extra
	assert.Len(t, result.Fields, 3)

	fieldNames := make(map[string]bool)
	for _, field := range result.Fields {
		fieldNames[field.Name] = true
	}

	assert.True(t, fieldNames["id"])
	assert.True(t, fieldNames["name"])
	assert.True(t, fieldNames["extra"])
}

// Test various numeric types
func TestNumericTypes(t *testing.T) {
	type TestNumeric struct {
		Int8       int8       `json:"int8"`
		Int16      int16      `json:"int16"`
		Int32      int32      `json:"int32"`
		Uint8      uint8      `json:"uint8"`
		Uint16     uint16     `json:"uint16"`
		Uint32     uint32     `json:"uint32"`
		Uintptr    uintptr    `json:"uintptr"`
		Float32    float32    `json:"float32"`
		Complex64  complex64  `json:"complex64"`
		Complex128 complex128 `json:"complex128"`
	}

	result := GetDocDefine(TestNumeric{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 10)

	for _, field := range result.Fields {
		switch field.Name {
		case "int8", "int16", "int32", "uint8", "uint16", "uint32", "uintptr":
			assert.Equal(t, TYPE_INT, field.Type)
		case "float32", "complex64", "complex128":
			assert.Equal(t, TYPE_FLOAT, field.Type)
		}
	}
}

// Test edge case with multi-level pointers
func TestMultiLevelPointers(t *testing.T) {
	type TestMultiPointer struct {
		SinglePtr **string  `json:"single_ptr"`
		DoublePtr ***string `json:"double_ptr"`
	}

	result := GetDocDefine(TestMultiPointer{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 2)

	for _, field := range result.Fields {
		assert.True(t, field.CanNull)
		// Multi-level pointers are handled as objects in parseType
		assert.Equal(t, TYPE_OBJECT, field.Type)
	}
}

// Test to cover remaining code paths
func TestRemainingCodePaths(t *testing.T) {
	// Test with a struct that has a field with a complex binding tag
	type TestComplexBinding struct {
		Field1 string `binding:"required,min=1,max=100"`
		Field2 string `binding:"omitempty"`
	}

	result := GetDocDefine(TestComplexBinding{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 2)

	for _, field := range result.Fields {
		if field.Name == "Field1" {
			assert.True(t, field.Required)
		}
	}

	// Test with a struct that has fields with various json tag combinations
	type TestComplexJSON struct {
		Field1 string `json:"field1,omitempty,string"`
		Field2 string `json:",omitempty"`
		Field3 string `json:"custom_name,string"`
	}

	result3 := GetDocDefine(TestComplexJSON{})
	assert.NotNil(t, result3)
	assert.Len(t, result3.Fields, 3)

	// Just verify the fields exist
	fieldNames := make(map[string]bool)
	for _, field := range result3.Fields {
		fieldNames[field.Name] = true
	}

	assert.True(t, fieldNames["field1"])
	// Empty json name should use field name - Field2 has json:",omitempty" so name should be "Field2"
	assert.True(t, fieldNames["Field2"], "Should have Field2")
	assert.True(t, fieldNames["custom_name"])
}

// Test to achieve 100% coverage by hitting the empty switch case
func TestNullTypeSwitchCase(t *testing.T) {
	// Create a struct with a field that has one of the null type names
	// but is not actually a pointer (to hit the first case in the switch)
	type NullString struct {
		Value string
	}

	type TestStruct struct {
		Field NullString `json:"field"`
	}

	result := GetDocDefine(TestStruct{})
	assert.NotNil(t, result)
	assert.Len(t, result.Fields, 1)

	field := result.Fields[0]
	assert.Equal(t, "field", field.Name)
	assert.Equal(t, TYPE_OBJECT, field.Type)
	assert.True(t, field.CanNull) // Should be true because of the "Null" prefix
}
