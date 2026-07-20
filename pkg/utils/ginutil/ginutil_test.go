package ginutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestParamID_valid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	id, ok := ParamID(c, "id")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if id != 42 {
		t.Fatalf("want id=42, got %d", id)
	}
}

func TestParamID_invalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	_, ok := ParamID(c, "id")
	if ok {
		t.Fatal("expected ok=false")
	}
	if w.Code != 400 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestParamID_largeNumber(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "999999999999"}}

	_, ok := ParamID(c, "id")
	if !ok {
		t.Fatal("expected ok=true for large number")
	}
}

func TestParamID_negative(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	_, ok := ParamID(c, "id")
	if ok {
		t.Fatal("expected ok=false for negative")
	}
}

func TestQueryPage_defaults(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?page=0&size=0", nil)

	page, size := QueryPage(c, 100)
	if page != 1 || size != 20 {
		t.Fatalf("page=%d size=%d", page, size)
	}
}

func TestQueryPage_customValues(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?page=3&size=50", nil)

	page, size := QueryPage(c, 100)
	if page != 3 || size != 50 {
		t.Fatalf("page=%d size=%d", page, size)
	}
}

func TestQueryPage_maxSize(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?page=1&size=200", nil)

	_, size := QueryPage(c, 100)
	if size != 100 {
		t.Fatalf("size should be clamped to 100, got %d", size)
	}
}

func TestQueryPage_noParams(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	page, size := QueryPage(c, 100)
	if page != 1 || size != 20 {
		t.Fatalf("page=%d size=%d", page, size)
	}
}

func TestBindJSON_valid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"test","value":123}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	var dest struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	ok := BindJSON(c, &dest)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if dest.Name != "test" || dest.Value != 123 {
		t.Fatalf("unexpected dest: %+v", dest)
	}
}

func TestBindJSON_invalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	var dest struct{}
	ok := BindJSON(c, &dest)
	if ok {
		t.Fatal("expected ok=false")
	}
	if w.Code != 400 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestRequireAuthTenant_authorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("auth.tenantId", uint(42))

	tid, ok := RequireAuthTenant(c)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if tid != 42 {
		t.Fatalf("want tid=42, got %d", tid)
	}
}

func TestRequireAuthTenant_unauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	_, ok := RequireAuthTenant(c)
	if ok {
		t.Fatal("expected ok=false")
	}
	if w.Code != 401 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestWriteGORMError_nil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	result := WriteGORMError(c, nil, "")
	if result {
		t.Fatal("nil error should return false")
	}
}

func TestWriteGORMError_notFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	result := WriteGORMError(c, gorm.ErrRecordNotFound, "")
	if !result {
		t.Fatal("expected true")
	}
	if w.Code != 404 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestWriteGORMError_notFoundCustomMsg(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	result := WriteGORMError(c, gorm.ErrRecordNotFound, "custom not found")
	if !result {
		t.Fatal("expected true")
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["msg"] != "custom not found" {
		t.Fatalf("msg=%v", resp["msg"])
	}
}

func TestWriteGORMError_internal(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	result := WriteGORMError(c, gorm.ErrInvalidDB, "ignored")
	if !result {
		t.Fatal("expected true")
	}
	if w.Code != 500 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestWriteInternalError_nil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	result := WriteInternalError(c, nil)
	if result {
		t.Fatal("nil error should return false")
	}
}

func TestWriteInternalError_error(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	result := WriteInternalError(c, gorm.ErrInvalidDB)
	if !result {
		t.Fatal("expected true")
	}
	if w.Code != 500 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestWriteAppError_nil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	WriteAppError(c, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("nil should not write status, got %d", w.Code)
	}
}

func TestPageSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	PageSuccess(c, []string{"a", "b"}, 2, 1, 20)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["msg"] != "success" {
		t.Fatalf("msg=%v", resp["msg"])
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("data should be map, got %T", resp["data"])
	}
	if data["total"] != float64(2) {
		t.Fatalf("total=%v", data["total"])
	}
}

func TestUploadURL_relative(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "http://localhost:7072/api/test", nil)
	c.Request.Host = "localhost:7072"

	url := UploadURL(c, "/test/recording.wav")
	if url == "" {
		t.Fatal("url should not be empty")
	}
}
