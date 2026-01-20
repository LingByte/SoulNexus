package LingEcho

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	DefaultQueryLimit = 102400 // 100k
)

const (
	FilterOpIsNot          = "is not"
	FilterOpEqual          = "="
	FilterOpNotEqual       = "<>"
	FilterOpIn             = "in"
	FilterOpNotIn          = "not_in"
	FilterOpGreater        = ">"
	FilterOpGreaterOrEqual = ">="
	FilterOpLess           = "<"
	FilterOpLessOrEqual    = "<="
	FilterOpLike           = "like"
	FilterOpBetween        = "between"
)

const (
	OrderOpDesc = "desc"
	OrderOpAsc  = "asc"
)

const (
	GET    = 1 << 1
	CREATE = 1 << 2
	EDIT   = 1 << 3
	DELETE = 1 << 4
	QUERY  = 1 << 5
)

type GetDB func(c *gin.Context, isCreate bool) *gorm.DB // designed for group
type PrepareQuery func(db *gorm.DB, c *gin.Context) (*gorm.DB, *QueryForm, error)

type (
	BeforeCreateFunc      func(db *gorm.DB, ctx *gin.Context, vptr any) error
	BeforeDeleteFunc      func(db *gorm.DB, ctx *gin.Context, vptr any) error
	BeforeUpdateFunc      func(db *gorm.DB, ctx *gin.Context, vptr any, vals map[string]any) error
	BeforeRenderFunc      func(db *gorm.DB, ctx *gin.Context, vptr any) (any, error)
	BeforeQueryRenderFunc func(db *gorm.DB, ctx *gin.Context, r *QueryResult) (any, error)
)

type QueryView struct {
	Path    string `json:"path"`
	Method  string `json:"method"`
	Desc    string `json:"desc"`
	Prepare PrepareQuery
}

type WebObjectPrimaryField struct {
	IsPrimary bool
	Name      string
	Kind      reflect.Kind
	JSONName  string
}

type WebObject struct {
	Model             any
	Group             string
	Name              string
	Desc              string
	AuthRequired      bool
	Editables         []string
	Filterables       []string
	Orderables        []string
	Searchables       []string
	GetDB             GetDB
	PrepareQuery      PrepareQuery
	BeforeCreate      BeforeCreateFunc
	BeforeUpdate      BeforeUpdateFunc
	BeforeDelete      BeforeDeleteFunc
	BeforeRender      BeforeRenderFunc
	BeforeQueryRender BeforeQueryRenderFunc

	Views        []QueryView
	AllowMethods int

	primaryKeys []WebObjectPrimaryField
	uniqueKeys  []WebObjectPrimaryField
	tableName   string

	// Model type
	modelElem reflect.Type
	// Map json tag to struct field name. such as:
	// UUID string `json:"id"` => {"id" : "UUID"}
	jsonToFields map[string]string
	// Map json tag to field kind. such as:
	// UUID string `json:"id"` => {"id": string}
	jsonToKinds map[string]reflect.Kind
}

type Filter struct {
	isTimeType bool   `json:"-"`
	Name       string `json:"name"`
	Op         string `json:"op"`
	Value      any    `json:"value"`
}

type Order struct {
	Name string `json:"name"`
	Op   string `json:"op"`
}

type QueryForm struct {
	Pos          int      `json:"pos"`
	Limit        int      `json:"limit"`
	Keyword      string   `json:"keyword,omitempty"`
	Filters      []Filter `json:"filters,omitempty"`
	Orders       []Order  `json:"orders,omitempty"`
	ForeignMode  bool     `json:"foreign"` // for foreign key
	ViewFields   []string `json:"-"`       // for view
	searchFields []string `json:"-"`       // for keyword
}

type QueryResult struct {
	TotalCount int    `json:"total,omitempty"`
	Pos        int    `json:"pos,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Keyword    string `json:"keyword,omitempty"`
	Items      []any  `json:"items"`
}

// GetQuery return the combined filter SQL statement.
// such as "age >= ?", "name IN ?".
func (f *Filter) GetQuery() string {
	var op string
	switch f.Op {
	case FilterOpIsNot:
		op = "IS NOT"
	case FilterOpEqual:
		op = "="
	case FilterOpNotEqual:
		op = "<>"
	case FilterOpIn:
		op = "IN"
	case FilterOpNotIn:
		op = "NOT IN"
	case FilterOpGreater:
		op = ">"
	case FilterOpGreaterOrEqual:
		op = ">="
	case FilterOpLess:
		op = "<"
	case FilterOpLessOrEqual:
		op = "<="
	case FilterOpLike:
		op = "LIKE"
	case FilterOpBetween:
		op = "BETWEEN"
		return fmt.Sprintf("`%s` BETWEEN ? AND ?", f.Name)
	}

	if op == "" {
		return ""
	}

	return fmt.Sprintf("`%s` %s ?", f.Name, op)
}

// GetQuery return the combined order SQL statement.
// such as "id DESC".
func (f *Order) GetQuery() string {
	if f.Op == OrderOpDesc {
		return f.Name + " DESC"
	}
	return f.Name + " ASC"
}

func (obj *WebObject) RegisterObject(r *gin.RouterGroup) error {
	if err := obj.Build(); err != nil {
		return err
	}

	p := obj.Name
	allowMethods := obj.AllowMethods
	if allowMethods == 0 {
		allowMethods = GET | CREATE | EDIT | DELETE | QUERY
	}

	primaryKeyPath := obj.BuildPrimaryPath(p)
	if allowMethods&GET != 0 {
		r.GET(primaryKeyPath, func(c *gin.Context) {
			handleGetObject(c, obj)
		})
	}
	if allowMethods&CREATE != 0 {
		r.PUT(p, func(c *gin.Context) {
			handleCreateObject(c, obj)
		})
	}
	if allowMethods&EDIT != 0 {
		r.PATCH(primaryKeyPath, func(c *gin.Context) {
			handleEditObject(c, obj)
		})
	}

	if allowMethods&DELETE != 0 {
		r.DELETE(primaryKeyPath, func(c *gin.Context) {
			handleDeleteObject(c, obj)
		})
	}

	if allowMethods&QUERY != 0 {
		r.POST(p, func(c *gin.Context) {
			handleQueryObject(c, obj, obj.PrepareQuery)
		})
	}

	for i := 0; i < len(obj.Views); i++ {
		v := &obj.Views[i]
		if v.Path == "" {
			return errors.New("with invalid view")
		}
		if v.Method == "" {
			v.Method = http.MethodPost
		}
		r.Handle(v.Method, filepath.Join(p, v.Path), func(ctx *gin.Context) {
			handleQueryObject(ctx, obj, v.Prepare)
		})
	}

	return nil
}

func (obj *WebObject) BuildPrimaryPath(prefix string) string {
	var primaryKeyPath []string
	for _, v := range obj.uniqueKeys {
		primaryKeyPath = append(primaryKeyPath, ":"+v.JSONName)
	}
	return filepath.Join(prefix, filepath.Join(primaryKeyPath...))
}

func (obj *WebObject) getPrimaryValues(c *gin.Context) ([]string, error) {
	var result []string
	for _, field := range obj.uniqueKeys {
		v := c.Param(field.JSONName)
		if v == "" {
			return nil, fmt.Errorf("invalid primary: %s", field.JSONName)
		}
		result = append(result, v)
	}
	return result, nil
}

func (obj *WebObject) buildPrimaryCondition(db *gorm.DB, keys []string) *gorm.DB {
	var tx *gorm.DB
	for i := 0; i < len(obj.uniqueKeys); i++ {
		colName := obj.uniqueKeys[i].Name
		col := db.NamingStrategy.ColumnName(obj.tableName, colName)
		tx = db.Where(col, keys[i])
	}
	return tx
}

/*
Check Go type corresponds to JSON type.
- float64, for JSON numbers
- string, for JSON strings
- []any, for JSON arrays
- map[string]any, for JSON objects
- nil, for JSON null
*/
func (obj *WebObject) checkType(db *gorm.DB, key string, value any) (string, bool, error) {
	targetKind, ok := obj.jsonToKinds[key]
	if !ok {
		return "", false, nil
	}

	fieldName, ok := obj.jsonToFields[key]
	if !ok {
		return "", false, nil
	}

	valueKind := reflect.TypeOf(value).Kind()
	var result bool

	switch targetKind {
	case reflect.Struct, reflect.Slice: // time.Time, associated structures
		result = true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		result = valueKind == reflect.Float64
	default:
		result = targetKind == valueKind
	}

	fieldName = db.NamingStrategy.ColumnName(obj.tableName, fieldName)
	if !result {
		return fieldName, false, fmt.Errorf("%s type not match", key)
	}
	return fieldName, true, nil
}

func RegisterObject(r *gin.RouterGroup, obj *WebObject) error {
	return obj.RegisterObject(r)
}

func RegisterObjects(r *gin.RouterGroup, objs []WebObject) {
	for idx := range objs {
		obj := &objs[idx]
		err := obj.RegisterObject(r)
		if err != nil {
			log.Fatalf("RegisterObject [%s] fail %v\n", obj.Name, err)
		}
	}
}

// Build fill the properties of obj.
func (obj *WebObject) Build() error {
	rt := reflect.TypeOf(obj.Model)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	obj.modelElem = rt
	obj.tableName = obj.modelElem.Name()

	if obj.Name == "" {
		obj.Name = strings.ToLower(obj.tableName)
	}

	obj.jsonToFields = make(map[string]string)
	obj.jsonToKinds = make(map[string]reflect.Kind)
	obj.parseFields(obj.modelElem)

	if obj.primaryKeys != nil {
		obj.uniqueKeys = obj.primaryKeys
	}

	if len(obj.uniqueKeys) <= 0 && len(obj.primaryKeys) <= 0 {
		return fmt.Errorf("%s not has primaryKey", obj.Name)
	}
	return nil
}

// parseFields parse the following properties according to struct tag:
// - jsonToFields, jsonToKinds, primaryKeyName, primaryKeyJsonName
func (obj *WebObject) parseFields(rt reflect.Type) {
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)

		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			obj.parseFields(f.Type)
			continue
		}

		jsonTag := strings.TrimSpace(strings.Split(f.Tag.Get("json"), ",")[0])
		if jsonTag == "" {
			obj.jsonToFields[f.Name] = f.Name

			kind := f.Type.Kind()
			if kind == reflect.Ptr {
				kind = f.Type.Elem().Kind()
			}
			obj.jsonToKinds[f.Name] = kind
		} else if jsonTag != "-" {
			obj.jsonToFields[jsonTag] = f.Name

			kind := f.Type.Kind()
			if kind == reflect.Ptr {
				kind = f.Type.Elem().Kind()
			}
			obj.jsonToKinds[jsonTag] = kind
		}

		gormTag := strings.ToLower(f.Tag.Get("gorm"))
		if gormTag == "-" {
			continue
		}
		pkField := WebObjectPrimaryField{
			Name:      f.Name,
			JSONName:  strings.Split(jsonTag, ",")[0],
			Kind:      f.Type.Kind(),
			IsPrimary: strings.Contains(gormTag, "primarykey"),
		}

		if pkField.JSONName == "" {
			pkField.JSONName = pkField.Name
		}

		if pkField.IsPrimary {
			obj.primaryKeys = append(obj.primaryKeys, pkField)
		} else if strings.Contains(gormTag, "unique") {
			obj.uniqueKeys = append(obj.uniqueKeys, pkField)
		}
	}
}

func GetDbConnection(c *gin.Context, objFn GetDB, isCreate bool) (tx *gorm.DB) {
	if objFn != nil {
		tx = objFn(c, isCreate)
	} else {
		tx = c.MustGet(constants.DbField).(*gorm.DB)
	}
	return tx.Session(&gorm.Session{})
}

func handleGetObject(c *gin.Context, obj *WebObject) {
	keys, err := obj.getPrimaryValues(c)
	if err != nil {
		AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	db := GetDbConnection(c, obj.GetDB, false)
	// the real name of the primaryKey column
	val := reflect.New(obj.modelElem).Interface()
	result := obj.buildPrimaryCondition(db, keys).Take(&val)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			AbortWithJSONError(c, http.StatusNotFound, errors.New("not found"))
		} else {
			AbortWithJSONError(c, http.StatusInternalServerError, result.Error)
		}
		return
	}

	if obj.BeforeRender != nil {
		rr, err := obj.BeforeRender(db, c, val)
		if err != nil {
			AbortWithJSONError(c, http.StatusInternalServerError, err)
			return
		}

		if c.Writer.Written() || c.Writer.Status() != http.StatusOK {
			// if body has written, return
			return
		}

		if rr != nil {
			val = rr
		}
	}

	c.JSON(http.StatusOK, val)
}

func handleCreateObject(c *gin.Context, obj *WebObject) {
	val := reflect.New(obj.modelElem).Interface()

	if c.Request.ContentLength > 0 {
		if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
			if err := c.BindJSON(&val); err != nil {
				AbortWithJSONError(c, http.StatusBadRequest, err)
				return
			}
		}
	}

	db := GetDbConnection(c, obj.GetDB, true)
	if obj.BeforeCreate != nil {
		if err := obj.BeforeCreate(db, c, val); err != nil {
			AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	result := db.Create(val)
	if result.Error != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, result.Error)
		return
	}

	c.JSON(http.StatusOK, val)
}

func handleEditObject(c *gin.Context, obj *WebObject) {
	keys, err := obj.getPrimaryValues(c)
	if err != nil {
		AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	var inputVals map[string]any
	if err := c.BindJSON(&inputVals); err != nil {
		AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	db := GetDbConnection(c, obj.GetDB, false)

	var vals map[string]any = map[string]any{}

	// can't edit primaryKey
	for _, k := range obj.uniqueKeys {
		delete(inputVals, k.JSONName)
	}

	for k, v := range inputVals {
		if v == nil {
			continue
		}

		fieldName, ok, err := obj.checkType(db, k, v)
		if err != nil {
			AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("%s type not match", k))
			return
		}
		if !ok { // ignore invalid field
			continue
		}
		vals[fieldName] = v
	}

	if len(obj.Editables) > 0 {
		stripVals := make(map[string]any)
		for _, k := range obj.Editables {
			k = db.NamingStrategy.ColumnName(obj.tableName, k)
			if v, ok := vals[k]; ok {
				stripVals[k] = v
			}
		}
		vals = stripVals
	} else {
		vals = map[string]any{}
	}

	if len(vals) == 0 {
		AbortWithJSONError(c, http.StatusBadRequest, errors.New("not changed"))
		return
	}
	db = obj.buildPrimaryCondition(db.Model(obj.Model), keys)

	if obj.BeforeUpdate != nil {
		val := reflect.New(obj.modelElem).Interface()
		tx := db.Session(&gorm.Session{})
		if err := tx.First(val).Error; err != nil {
			AbortWithJSONError(c, http.StatusNotFound, errors.New("not found"))
			return
		}
		if err := obj.BeforeUpdate(db, c, val, inputVals); err != nil {
			AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	result := db.Updates(vals)
	if result.Error != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, result.Error)
		return
	}

	c.JSON(http.StatusOK, true)
}

func handleDeleteObject(c *gin.Context, obj *WebObject) {
	keys, err := obj.getPrimaryValues(c)
	if err != nil {
		AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	db := GetDbConnection(c, obj.GetDB, false)
	val := reflect.New(obj.modelElem).Interface()

	r := obj.buildPrimaryCondition(db, keys).Session(&gorm.Session{}).First(val)

	// for gorm delete hook, need to load rtcmedia first.
	if r.Error != nil {
		if errors.Is(r.Error, gorm.ErrRecordNotFound) {
			AbortWithJSONError(c, http.StatusNotFound, errors.New("not found"))
		} else {
			AbortWithJSONError(c, http.StatusInternalServerError, r.Error)
		}
		return
	}

	if obj.BeforeDelete != nil {
		if err := obj.BeforeDelete(db, c, val); err != nil {
			AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	r = db.Delete(val)
	if r.Error != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, r.Error)
		return
	}

	c.JSON(http.StatusOK, true)
}

func handleQueryObject(c *gin.Context, obj *WebObject, prepareQuery PrepareQuery) {
	if prepareQuery == nil {
		prepareQuery = DefaultPrepareQuery
	}
	db, form, err := prepareQuery(GetDbConnection(c, obj.GetDB, false), c)
	if err != nil {
		AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	namer := db.NamingStrategy

	// Use struct{} makes map like set.
	var filterFields = make(map[string]struct{})
	for _, k := range obj.Filterables {
		filterFields[k] = struct{}{}
	}

	if len(filterFields) > 0 {
		var stripFilters []Filter
		for i := 0; i < len(form.Filters); i++ {
			filter := form.Filters[i]
			// Struct must has this field.
			field, ok := obj.jsonToFields[filter.Name]
			if !ok {
				continue
			}
			if _, ok := filterFields[field]; !ok {
				continue
			}

			if f, ok := obj.modelElem.FieldByName(field); ok {
				var typeName string = f.Type.Name()
				if f.Type.Kind() == reflect.Ptr {
					typeName = f.Type.Elem().Name()
				}
				filter.isTimeType = typeName == "Time" || typeName == "NullTime" || typeName == "DeletedAt"
			}
			filter.Name = namer.ColumnName(obj.tableName, field)
			stripFilters = append(stripFilters, filter)
		}
		form.Filters = stripFilters
	} else {
		form.Filters = []Filter{}
	}

	var orderFields = make(map[string]struct{})
	for _, k := range obj.Orderables {
		orderFields[k] = struct{}{}
	}
	if len(orderFields) > 0 {
		var stripOrders []Order
		for i := 0; i < len(form.Orders); i++ {
			order := form.Orders[i]
			field, ok := obj.jsonToFields[order.Name]
			if !ok {
				continue
			}
			if _, ok := orderFields[field]; !ok {
				continue
			}
			order.Name = namer.ColumnName(obj.tableName, order.Name)
			stripOrders = append(stripOrders, order)
		}
		form.Orders = stripOrders
	} else {
		form.Orders = []Order{}
	}

	if form.Keyword != "" {
		form.searchFields = []string{}
		for _, v := range obj.Searchables {
			form.searchFields = append(form.searchFields, namer.ColumnName(obj.tableName, v))
		}
	}

	if len(form.ViewFields) > 0 {
		var stripViewFields []string
		for _, v := range form.ViewFields {
			stripViewFields = append(stripViewFields, namer.ColumnName(obj.tableName, v))
		}
		form.ViewFields = stripViewFields
	}

	r, err := obj.queryObjects(db, c, form)
	if err != nil {
		AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if obj.BeforeQueryRender != nil {
		obj, err := obj.BeforeQueryRender(db, c, &r)
		if err != nil {
			AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}

		if c.Writer.Written() || c.Writer.Status() != http.StatusOK {
			// if body has written, return
			return
		}

		if obj != nil {
			c.JSON(http.StatusOK, obj)
			return
		}
	}
	c.JSON(http.StatusOK, r)
}

func castTime(value any) any {
	if tv, ok := value.(string); ok {
		for _, tf := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02", time.RFC1123} {
			t, err := time.Parse(tf, tv)
			if err == nil {
				return t
			}
		}
	}
	return value
}

func (obj *WebObject) queryObjects(db *gorm.DB, ctx *gin.Context, form *QueryForm) (r QueryResult, err error) {
	tblName := db.NamingStrategy.TableName(obj.tableName)

	for _, v := range form.Filters {
		if q := v.GetQuery(); q != "" {
			if v.Op == FilterOpLike {
				if kws, ok := v.Value.([]any); ok {
					qs := []string{}
					for _, kw := range kws {
						k := fmt.Sprintf("\"%%%s%%\"", strings.ReplaceAll(kw.(string), "\"", "\\\""))
						q := fmt.Sprintf("`%s`.`%s` LIKE %s", tblName, v.Name, k)
						qs = append(qs, q)
					}
					db = db.Where(strings.Join(qs, " OR "))
				} else {
					db = db.Where(fmt.Sprintf("`%s`.%s", tblName, q), fmt.Sprintf("%%%s%%", v.Value))
				}
			} else if v.Op == FilterOpBetween {
				vt := reflect.ValueOf(v.Value)
				if vt.Kind() != reflect.Slice && vt.Len() != 2 {
					return r, fmt.Errorf("invalid between value, must be slice with 2 elements")
				}

				leftValue := vt.Index(0).Interface()
				rightValue := vt.Index(1).Interface()
				if v.isTimeType {
					leftValue = castTime(leftValue)
					rightValue = castTime(rightValue)
				}
				db = db.Where(fmt.Sprintf("`%s`.%s", tblName, q), leftValue, rightValue)
			} else {
				if v.isTimeType {
					v.Value = castTime(v.Value)
				}
				db = db.Where(fmt.Sprintf("`%s`.%s", tblName, q), v.Value)
			}
		}
	}

	for _, v := range form.Orders {
		if q := v.GetQuery(); q != "" {
			db = db.Order(fmt.Sprintf("%s.%s", tblName, q))
		}
	}

	if form.Keyword != "" && len(form.searchFields) > 0 {
		var query []string
		for _, v := range form.searchFields {
			query = append(query, fmt.Sprintf("`%s`.`%s` LIKE @keyword", tblName, v))
		}
		searchKey := strings.Join(query, " OR ")
		db = db.Where(searchKey, sql.Named("keyword", "%"+form.Keyword+"%"))
	}

	if len(form.ViewFields) > 0 {
		db = db.Select(form.ViewFields)
	}

	r.Pos = form.Pos
	r.Limit = form.Limit
	r.Keyword = form.Keyword

	var c int64
	if err := db.Model(obj.Model).Count(&c).Error; err != nil {
		return r, err
	}
	if c <= 0 {
		return r, nil
	}
	r.TotalCount = int(c)

	vals := reflect.New(reflect.SliceOf(obj.modelElem))
	result := db.Offset(form.Pos).Limit(form.Limit).Find(vals.Interface())
	if result.Error != nil {
		return r, result.Error
	}

	r.Items = make([]any, 0, vals.Elem().Len())
	for i := 0; i < vals.Elem().Len(); i++ {
		modelObj := vals.Elem().Index(i).Addr().Interface()
		if obj.BeforeRender != nil {
			rr, err := obj.BeforeRender(db, ctx, modelObj)
			if err != nil {
				return r, err
			}
			if rr != nil {
				// if BeforeRender return not nil, then use it as result
				modelObj = rr
			}
		}
		r.Items = append(r.Items, modelObj)
	}
	r.Pos += int(len(r.Items))
	return r, nil
}

// DefaultPrepareQuery return default QueryForm.
func DefaultPrepareQuery(db *gorm.DB, c *gin.Context) (*gorm.DB, *QueryForm, error) {
	var form QueryForm
	if c.Request.ContentLength > 0 {
		if err := c.BindJSON(&form); err != nil {
			return nil, nil, err
		}
	}

	if form.Pos < 0 {
		form.Pos = 0
	}
	if form.Limit <= 0 || form.Limit > DefaultQueryLimit {
		form.Limit = DefaultQueryLimit
	}

	return db, &form, nil
}

type ErrorWithCode interface {
	StatusCode() int
}

func AbortWithJSONError(c *gin.Context, code int, err error) {
	var errWithFileNum error = err
	if log.Flags()&(log.Lshortfile|log.Llongfile) != 0 {
		var ok bool
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "???"
			line = 0
		}
		pos := strings.LastIndex(file, "/")
		if log.Flags()&log.Lshortfile != 0 && pos >= 0 {
			file = file[1+pos:]
		}
		errWithFileNum = fmt.Errorf("%s:%d: %v", file, line, err)
	}
	c.Error(errWithFileNum)

	if e, ok := err.(ErrorWithCode); ok {
		code = e.StatusCode()
	}

	if c.IsAborted() {
		c.JSON(code, gin.H{"error": err.Error()})
		response.Fail(c, err.Error(), nil)
	} else {
		response.AbortWithStatusJSON(c, code, err)
	}
}
