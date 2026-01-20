package utils

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
)

// 定义测试请求结构体
type TestRequest struct {
	Name string `json:"name" binding:"required"`
	Age  int    `json:"age"`
}

func TestBindSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		var req TestRequest
		err := c.ShouldBindJSON(&req)
		assert.NoError(t, err)
		RenderJSON(c, 200, gin.H{"msg": "ok", "name": req.Name})
	})

	body := `{"name": "Alice", "age": 30}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Alice"`)
}

func TestBindMissingField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		var req TestRequest
		err := c.ShouldBindJSON(&req)
		if err != nil {
			RenderJSON(c, 400, gin.H{"error": err.Error()})
			return
		}
		RenderJSON(c, 200, gin.H{"msg": "ok"})
	})

	body := `{"age": 20}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), `"error"`)
}

func TestUnknownFieldRejection(t *testing.T) {
	EnableDecoderDisallowUnknownFields = true // 开启严格模式
	defer func() { EnableDecoderDisallowUnknownFields = false }()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		var req TestRequest
		err := c.ShouldBindJSON(&req)
		if err != nil {
			RenderJSON(c, 400, gin.H{"error": err.Error()})
			return
		}
		RenderJSON(c, 200, gin.H{"msg": "ok"})
	})

	body := `{"name": "Bob", "unknown": "field"}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), `unknown field`)
}

func TestBindBodySuccess(t *testing.T) {
	jsonData := []byte(`{"name":"Charlie", "age":25}`)
	var req TestRequest
	err := CarrotJsonBinding{}.BindBody(jsonData, &req)
	assert.NoError(t, err)
	assert.Equal(t, "Charlie", req.Name)
}

func TestBindBodyInvalidJSON(t *testing.T) {
	jsonData := []byte(`{"name":`) // malformed JSON
	var req TestRequest
	err := CarrotJsonBinding{}.BindBody(jsonData, &req)
	assert.Error(t, err)
}

func TestWriteJSONInvalidData(t *testing.T) {
	rec := httptest.NewRecorder()
	err := WriteJSON(rec, map[any]any{1: "invalid"}) // will cause marshal error
	assert.Error(t, err)
}

func TestBindInvalidRequest(t *testing.T) {
	var req TestRequest
	err := CarrotJsonBinding{}.Bind(nil, &req)
	assert.Error(t, err)
}

func TestWriteContentTypeHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	carrot := CarrotJSON{}
	carrot.WriteContentType(rec)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestRenderJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	c := CarrotJSON{Data: gin.H{"hello": "world"}}
	err := c.Render(rec)
	assert.NoError(t, err)
	assert.Contains(t, rec.Body.String(), `"hello":"world"`)
}

func TestValidateNilValidator(t *testing.T) {
	original := binding.Validator
	binding.Validator = nil
	defer func() { binding.Validator = original }()

	err := validate(&TestRequest{Name: "OK"})
	assert.NoError(t, err)
}
