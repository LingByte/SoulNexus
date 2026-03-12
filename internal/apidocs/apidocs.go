package apidocs

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	_ "embed"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

//go:embed  apidocs.html
var apiDocHTML string

type OptionFunc func(*UriDoc)

const (
	TYPE_DATE    = "date"
	TYPE_STRING  = "string"
	TYPE_INT     = "int"
	TYPE_FLOAT   = "float"
	TYPE_BOOLEAN = "boolean"
	TYPE_OBJECT  = "object"
	TYPE_MAP     = "map"
)

type DocField struct {
	FieldName string     `json:"-"`
	Name      string     `json:"name"`
	Desc      string     `json:"desc,omitempty"`
	Type      string     `json:"type,omitempty"`
	Default   any        `json:"default,omitempty"`
	Required  bool       `json:"required,omitempty"`
	CanNull   bool       `json:"canNull,omitempty"`
	IsArray   bool       `json:"isArray,omitempty"`
	IsPrimary bool       `json:"isPrimary,omitempty"`
	Fields    []DocField `json:"fields,omitempty"`
}

type WebObjectDoc struct {
	Group        string     `json:"group"`
	Path         string     `json:"path"`
	Desc         string     `json:"desc,omitempty"`
	AuthRequired bool       `json:"authRequired,omitempty"`
	AllowMethods []string   `json:"allowMethods,omitempty"`
	Fields       []DocField `json:"fields,omitempty"` // Request Body
	Filters      []string   `json:"filters,omitempty"`
	Orders       []string   `json:"orders,omitempty"`
	Searches     []string   `json:"searches,omitempty"`
	Editables    []string   `json:"editables,omitempty"`
	Views        []UriDoc   `json:"views,omitempty"`
}

type UriDoc struct {
	MethodRef    any       `json:"-"` // just for quick jump to method
	Group        string    `json:"group"`
	Path         string    `json:"path"`
	Summary      string    `json:"summary"`
	Desc         string    `json:"desc,omitempty"`
	AuthRequired bool      `json:"authRequired,omitempty"`
	Method       string    `json:"method"` // "GET" "POST" "DELETE" "PUT" "PATCH"
	Request      *DocField `json:"request"`
	Response     *DocField `json:"response"`
}

func RegisterHandler(prefix string, r *gin.Engine, uriDocs []UriDoc, objDocs []WebObjectDoc, db *gorm.DB) {
	prefix = strings.TrimSuffix(prefix, "/")
	r.GET(prefix+".json", func(ctx *gin.Context) {
		ctx.Set(constants.DbField, db)
		docs := map[string]any{
			"uris": uriDocs,
			"objs": objDocs,
			"site": LingEcho.GetRenderPageContext(ctx),
		}
		ctx.JSON(http.StatusOK, docs)
	})

	r.GET(prefix, func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(apiDocHTML))
	})
}

func GetDocDefine(obj any) *DocField {
	if obj == nil {
		return nil
	}
	rt := reflect.TypeOf(obj)
	f := parseDocField(rt, "", nil)
	return &f
}

func GetWebObjectDocDefine(prefix string, obj LingEcho.WebObject) WebObjectDoc {
	doc := WebObjectDoc{
		Group:        obj.Group,
		Path:         filepath.Join(prefix, obj.Name),
		Desc:         obj.Desc,
		AuthRequired: obj.AuthRequired,
		Filters:      obj.Filterables,
		Orders:       obj.Orderables,
		Searches:     obj.Searchables,
	}
	allowMethods := obj.AllowMethods
	if obj.AllowMethods == 0 {
		allowMethods = LingEcho.GET | LingEcho.CREATE | LingEcho.EDIT | LingEcho.DELETE | LingEcho.QUERY
	}

	if allowMethods&LingEcho.GET != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "GET")
	}
	if allowMethods&LingEcho.CREATE != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "CREATE")
	}
	if allowMethods&LingEcho.EDIT != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "EDIT")
	}
	if allowMethods&LingEcho.DELETE != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "DELETE")
	}
	if allowMethods&LingEcho.QUERY != 0 {
		doc.AllowMethods = append(doc.AllowMethods, "QUERY")
	}

	doc.Fields = GetDocDefine(obj.Model).Fields
	allFields := []string{}
	for _, f := range doc.Fields {
		allFields = append(allFields, f.Name)
	}

	if len(obj.Editables) == 0 {
		doc.Editables = allFields
	} else {
		for _, ef := range obj.Editables {
			for _, f := range doc.Fields {
				if ef == f.FieldName {
					doc.Editables = append(doc.Editables, f.Name)
				}
			}
		}
	}

	for _, v := range obj.Views {
		doc.Views = append(doc.Views, UriDoc{
			Path:   filepath.Join(doc.Path, v.Path),
			Method: v.Method,
			Desc:   v.Desc,
		})
	}

	return doc
}

// parseDocField convert StructField Type to DocFiled.
func parseDocField(rt reflect.Type, name string, stacks []string) (val DocField) {
	val.Name = name
	val.Type = parseType(rt)

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		val.CanNull = true
	}

	if strings.HasPrefix(rt.Name(), "Null") {
		val.CanNull = true
	}

	if rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice {
		rt = rt.Elem()
		val.IsArray = true
	}

	switch rt.Name() {
	case "NullTime", "NullBool", "NullString", "NullByte", "NullInt16",
		"NullInt32", "NullInt64", "NullFloat32", "NullFloat64":
		// These types are handled as objects, continue processing
	case "Time", "DeletedAt":
		return val
	}

	if rt.Kind() != reflect.Struct {
		return val
	}

	val.Type = TYPE_OBJECT

	for _, v := range stacks {
		if rt.Name() == v {
			return val
		}
	}

	stacks = append(stacks, rt.Name())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i) // StructField
		jsonTag := f.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		// process embeded struct
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			embedRT := parseDocField(f.Type, "", stacks)
			val.Fields = append(val.Fields, embedRT.Fields...)
			continue
		}

		var name = f.Name
		if jsonTag != "" {
			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName != "" {
				name = jsonName
			}
		}

		fieldRT := parseDocField(f.Type, name, stacks)
		fieldRT.FieldName = f.Name
		fieldRT.Desc = f.Tag.Get("comment")

		if strings.Contains(f.Tag.Get("binding"), "required") {
			fieldRT.Required = true
		}

		if strings.Contains(jsonTag, "omitempty") {
			fieldRT.CanNull = true
		}

		if strings.Contains(f.Tag.Get("gorm"), "primary") {
			fieldRT.IsPrimary = true
		}

		val.Fields = append(val.Fields, fieldRT)
	}
	return val
}

// parseType return type string according to reflect.Type.
func parseType(rt reflect.Type) string {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	// Multi-Level Pointers
	if rt.Kind() == reflect.Ptr {
		return TYPE_OBJECT
	}

	switch rt.Name() {
	case "NullTime", "Time", "DeletedAt":
		return TYPE_DATE
	}

	switch rt.Kind() {
	case reflect.Array, reflect.Slice:
		val := rt.Elem().Kind().String()
		if val == "struct" || val == "ptr" {
			val = TYPE_OBJECT
		}
		return val
	case reflect.String:
		return TYPE_STRING
	case reflect.Bool:
		return TYPE_BOOLEAN
	case reflect.Map:
		return TYPE_MAP
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return TYPE_INT
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return TYPE_FLOAT
	}

	return ""
}
