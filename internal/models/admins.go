package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/inflection"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const KEY_ADMIN_DASHBOARD = "ADMIN_DASHBOARD"

type AdminBuildContext func(*gin.Context, map[string]any) map[string]any

type AdminQueryResult struct {
	TotalCount int              `json:"total,omitempty"`
	Pos        int              `json:"pos,omitempty"`
	Limit      int              `json:"limit,omitempty"`
	Keyword    string           `json:"keyword,omitempty"`
	Items      []map[string]any `json:"items"`
	objects    []any            `json:"-"`
}

// Access control
type AdminAccessCheck func(c *gin.Context, obj *AdminObject) error
type AdminActionHandler func(db *gorm.DB, c *gin.Context, obj any) (bool, any, error)
type AdminViewOnSite func(db *gorm.DB, c *gin.Context, obj any) string

type AdminSelectOption struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

type AdminAttribute struct {
	Default      any                 `json:"default,omitempty"`
	Choices      []AdminSelectOption `json:"choices,omitempty"`
	SingleChoice bool                `json:"singleChoice,omitempty"`
	Widget       string              `json:"widget,omitempty"`
	FilterWidget string              `json:"filterWidget,omitempty"`
	Help         string              `json:"help,omitempty"`
}
type AdminForeign struct {
	Path       string `json:"path"`
	Field      string `json:"field"`
	fieldName  string `json:"-"`
	foreignKey string `json:"-"`
}
type AdminValue struct {
	Value any    `json:"value"`
	Label string `json:"label,omitempty"`
}
type AdminIcon struct {
	Url string `json:"url,omitempty"`
	SVG string `json:"svg,omitempty"`
}

type AdminField struct {
	Placeholder string          `json:"placeholder,omitempty"` // Placeholder of the filed
	Label       string          `json:"label"`                 // Label of the filed
	NotColumn   bool            `json:"notColumn,omitempty"`   // Not a column
	Required    bool            `json:"required,omitempty"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Tag         string          `json:"tag,omitempty"`
	Attribute   *AdminAttribute `json:"attribute,omitempty"`
	CanNull     bool            `json:"canNull,omitempty"`
	IsArray     bool            `json:"isArray,omitempty"`
	Primary     bool            `json:"primary,omitempty"`
	Foreign     *AdminForeign   `json:"foreign,omitempty"`
	IsAutoID    bool            `json:"isAutoId,omitempty"`
	IsPtr       bool            `json:"isPtr,omitempty"`
	elemType    reflect.Type    `json:"-"`
	fieldName   string          `json:"-"`
}
type AdminScript struct {
	Src    string `json:"src"`
	Onload bool   `json:"onload,omitempty"`
}
type AdminAction struct {
	Path          string             `json:"path"`
	Name          string             `json:"name"`
	Label         string             `json:"label,omitempty"`
	Icon          string             `json:"icon,omitempty"`
	Class         string             `json:"class,omitempty"`
	WithoutObject bool               `json:"withoutObject"`
	Batch         bool               `json:"batch,omitempty"`
	Handler       AdminActionHandler `json:"-"`
}

type AdminObject struct {
	Model            any                       `json:"-"`
	Group            string                    `json:"group"`               // Group name
	Name             string                    `json:"name"`                // Name of the object
	Desc             string                    `json:"desc,omitempty"`      // Description
	Path             string                    `json:"path"`                // Path prefix
	Shows            []string                  `json:"shows"`               // Show fields
	Orders           []LingEcho.Order          `json:"orders"`              // Default orders of the object
	Editables        []string                  `json:"editables"`           // Editable fields
	Filterables      []string                  `json:"filterables"`         // Filterable fields
	Orderables       []string                  `json:"orderables"`          // Orderable fields, can override Orders
	Searchables      []string                  `json:"searchables"`         // Searchable fields
	Requireds        []string                  `json:"requireds,omitempty"` // Required fields
	PrimaryKeys      []string                  `json:"primaryKeys"`         // Primary keys name
	UniqueKeys       []string                  `json:"uniqueKeys"`          // Primary keys name
	PluralName       string                    `json:"pluralName"`
	Fields           []AdminField              `json:"fields"`
	EditPage         string                    `json:"editpage,omitempty"`
	ListPage         string                    `json:"listpage,omitempty"`
	Scripts          []AdminScript             `json:"scripts,omitempty"`
	Styles           []string                  `json:"styles,omitempty"`
	Permissions      map[string]bool           `json:"permissions,omitempty"`
	Actions          []AdminAction             `json:"actions,omitempty"`
	Icon             *AdminIcon                `json:"icon,omitempty"`
	Invisible        bool                      `json:"invisible,omitempty"`
	ViewOnSite       AdminViewOnSite           `json:"-"`
	Attributes       map[string]AdminAttribute `json:"-"` // Field's extra attributes
	AccessCheck      AdminAccessCheck          `json:"-"` // Access control function
	GetDB            LingEcho.GetDB            `json:"-"`
	BeforeCreate     LingEcho.BeforeCreateFunc `json:"-"`
	BeforeRender     LingEcho.BeforeRenderFunc `json:"-"`
	BeforeUpdate     LingEcho.BeforeUpdateFunc `json:"-"`
	BeforeDelete     LingEcho.BeforeDeleteFunc `json:"-"`
	tableName        string                    `json:"-"`
	modelElem        reflect.Type              `json:"-"`
	ignores          map[string]bool           `json:"-"`
	primaryKeyMaping map[string]string         `json:"-"`
}

// Returns all admin objects
func GetLingEchoAdminObjects() []AdminObject {

	superAccessCheck := func(c *gin.Context, obj *AdminObject) error {
		user := CurrentUser(c)
		if !user.IsSuperAdmin() {
			return errors.New("only superadmin can access")
		}
		return nil
	}

	iconUser, _ := LingEcho.EmbedStaticAssets.ReadFile("static/img/icon_user.svg")
	iconGroup, _ := LingEcho.EmbedStaticAssets.ReadFile("static/img/icon_group.svg")
	iconMembers, _ := LingEcho.EmbedStaticAssets.ReadFile("static/img/icon_members.svg")
	iconConfig, _ := LingEcho.EmbedStaticAssets.ReadFile("static/img/icon_config.svg")

	return []AdminObject{
		{
			Model:       &User{},
			Group:       "Settings",
			Name:        "User",
			Desc:        "Builtin user management system",
			Shows:       []string{"ID", "Email", "Username", "FirstName", "LastName", "IsStaff", "Role", "Enabled", "Activated", "UpdatedAt", "LastLogin", "LastLoginIP", "Source"},
			Editables:   []string{"Email", "Password", "Username", "FirstName", "LastName", "IsStaff", "Role", "Permissions", "Enabled", "Activated", "Profile", "Source"},
			Filterables: []string{"CreatedAt", "UpdatedAt", "Username", "IsStaff", "Role", "Enabled", "Activated "},
			Orderables:  []string{"CreatedAt", "UpdatedAt", "Enabled", "Activated"},
			Searchables: []string{"Username", "Email", "FirstName", "ListName"},
			Orders:      []LingEcho.Order{{"UpdatedAt", LingEcho.OrderOpDesc}},
			Icon:        &AdminIcon{SVG: string(iconUser)},
			AccessCheck: superAccessCheck,
			BeforeCreate: func(db *gorm.DB, c *gin.Context, obj any) error {
				user := obj.(*User)
				if user.Password != "" {
					user.Password = HashPassword(user.Password)
				}
				user.Source = "admin"
				return nil
			},
			BeforeUpdate: func(db *gorm.DB, c *gin.Context, obj any, vals map[string]any) error {
				user := obj.(*User)
				if dbUser, err := GetUserByEmail(db, user.Email); err == nil {
					if dbUser.Password != user.Password {
						user.Password = HashPassword(user.Password)
					}
				}
				return nil
			},
			Actions: []AdminAction{
				{
					Path:  "toggle_enabled",
					Name:  "Toggle enabled",
					Label: "Toggle user enabled/disabled",
					Handler: func(db *gorm.DB, c *gin.Context, obj any) (bool, any, error) {
						user := obj.(*User)
						err := UpdateUserFields(db, user, map[string]any{"Enabled": !user.Enabled})
						return false, user.Enabled, err
					},
				},
				{
					Path:  "toggle_staff",
					Name:  "Toggle staff",
					Label: "Toggle user is staff or not",
					Handler: func(db *gorm.DB, c *gin.Context, obj any) (bool, any, error) {
						user := obj.(*User)
						err := UpdateUserFields(db, user, map[string]any{"IsStaff": !user.IsStaff})
						return false, user.IsStaff, err
					},
				},
			},
			Attributes: map[string]AdminAttribute{
				"Password": {
					Widget: "password",
				},
			},
		},
		{
			Model:       &Group{},
			Group:       "Settings",
			Name:        "Group",
			Desc:        "A group describes a group of users. One user can be part of many groups and one group can have many users", //
			Shows:       []string{"ID", "Name", "Extra", "UpdatedAt", "CreatedAt"},
			Editables:   []string{"ID", "Name", "Extra", "UpdatedAt"},
			Orderables:  []string{"UpdatedAt"},
			Searchables: []string{"Name"},
			Requireds:   []string{"Name"},
			Icon:        &AdminIcon{SVG: string(iconGroup)},
			AccessCheck: superAccessCheck,
		},
		{
			Model:       &GroupMember{},
			Group:       "Settings",
			Name:        "GroupMember",
			Desc:        "Group members", //
			Shows:       []string{"ID", "User", "Group", "Role", "CreatedAt"},
			Filterables: []string{"Group", "Role", "CreatedAt"},
			Editables:   []string{"ID", "User", "Group", "Role"},
			Orderables:  []string{"CreatedAt"},
			Searchables: []string{"User", "Group"},
			Requireds:   []string{"User", "Group", "Role"},
			Icon:        &AdminIcon{SVG: string(iconMembers)},
			AccessCheck: superAccessCheck,
			Attributes: map[string]AdminAttribute{
				"Role": {
					Default: GroupRoleMember,
					Choices: []AdminSelectOption{{"Admin", GroupRoleAdmin}, {"Member", GroupRoleMember}},
				},
			},
		},
		{
			Model:       &utils.Config{},
			Group:       "Settings",
			Name:        "Config",
			Desc:        "System config with database backend, You can change it in admin page, and it will take effect immediately without restarting the server", //
			Shows:       []string{"Key", "Value", "Autoload", "Public", "Format", "Desc"},
			Editables:   []string{"Key", "Value", "Autoload", "Public", "Format", "Desc"},
			Filterables: []string{"Autoload", "Public"},
			Orderables:  []string{"Key"},
			Searchables: []string{"Key", "Value", "Desc"},
			Requireds:   []string{"Key", "Value"},
			Icon:        &AdminIcon{SVG: string(iconConfig)},
			AccessCheck: superAccessCheck,
		},
	}
}

func WithAdminAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := CurrentUser(ctx)
		if user == nil {
			db := ctx.MustGet(constants.DbField).(*gorm.DB)
			signUrl := utils.GetValue(db, constants.KEY_SITE_SIGNIN_URL)
			if signUrl == "" {
				LingEcho.AbortWithJSONError(ctx, http.StatusUnauthorized, errors.New("login required"))
			} else {
				ctx.Redirect(http.StatusFound, signUrl+"?next="+ctx.Request.URL.String())
				ctx.Abort()
			}
			return
		}

		// Validate user role before checking admin permissions
		if err := ValidateUserRole(user); err != nil {
			LingEcho.AbortWithJSONError(ctx, http.StatusForbidden, fmt.Errorf("invalid user role: %w", err))
			return
		}

		if !user.IsStaff && !user.IsAdmin() {
			LingEcho.AbortWithJSONError(ctx, http.StatusForbidden, errors.New("forbidden"))
			return
		}
		ctx.Next()
	}
}

func BuildAdminObjects(r *gin.RouterGroup, db *gorm.DB, objs []AdminObject) []*AdminObject {
	handledObjects := make([]*AdminObject, 0)
	exists := make(map[string]bool)
	for idx := range objs {
		obj := &objs[idx]
		err := obj.Build(db)
		if err != nil {
			logger.Warn("Build admin object fail, ignore" + obj.Group + obj.Name)
			continue
		}

		if _, ok := exists[obj.Path]; ok {
			logger.Warn("Ignore exist admin object" + obj.Group + obj.Name)
			continue
		}

		objr := r.Group(obj.Path)
		obj.Path = path.Join(r.BasePath(), obj.Path) + "/"
		for idx := range obj.Fields {
			f := &obj.Fields[idx]
			if f.Foreign == nil {
				continue
			}
			f.Foreign.Path = path.Join(r.BasePath(), f.Foreign.Path) + "/"
		}

		obj.RegisterAdmin(objr)
		handledObjects = append(handledObjects, obj)
	}
	return handledObjects
}

// InjectDB Inject database instance into Gin context
func InjectDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(constants.DbField, db)
		c.Next()
	}
}

// RegisterAdmins registers admin routes
func RegisterAdmins(r *gin.RouterGroup, db *gorm.DB, objs []AdminObject) {
	r.Use(InjectDB(db))
	r.Use(WithAdminAuth())

	handledObjects := BuildAdminObjects(r, db, objs)
	r.POST("/admin.json", func(ctx *gin.Context) {
		HandleAdminJson(ctx, handledObjects, func(ctx *gin.Context, m map[string]any) map[string]any {
			m["dashboard"] = utils.GetValue(db, KEY_ADMIN_DASHBOARD)
			return m
		})
	})
	r.GET("/*filepath", func(ctx *gin.Context) {
		staticAssets := ctx.MustGet(constants.AssetsField).(*LingEcho.CombineEmbedFS)
		name := ctx.Param("filepath")
		if name == "/" {
			var jsFiles []string
			var cssFiles []string
			dirs, err := staticAssets.ReadDir("admin")
			if err == nil {
				for _, dir := range dirs {
					if dir.IsDir() {
						continue
					}
					if strings.HasSuffix("admin/"+dir.Name(), ".css") {
						cssFiles = append(cssFiles, dir.Name())
					}
					if strings.HasSuffix("admin/"+dir.Name(), ".js") {
						jsFiles = append(jsFiles, dir.Name())
					}
				}
			}

			ctx.HTML(http.StatusOK, "app.html", gin.H{
				"Scripts":   jsFiles,
				"Styles":    cssFiles,
				"Dashboard": utils.GetValue(db, KEY_ADMIN_DASHBOARD),
				"Objects":   handledObjects,
			})
			return
		}
		ctx.FileFromFS(filepath.Join("admin", name), http.FS(staticAssets))
	})
}

func HandleAdminJson(c *gin.Context, objects []*AdminObject, buildContext AdminBuildContext) {
	var viewObjects []AdminObject
	for _, obj := range objects {
		if obj.AccessCheck != nil {
			err := obj.AccessCheck(c, obj)
			if err != nil {
				continue
			}
		}
		val := *obj
		viewObjects = append(viewObjects, val)
	}

	siteCtx := LingEcho.GetRenderPageContext(c)
	if buildContext != nil {
		siteCtx = buildContext(c, siteCtx)
	}

	// Add monitoring page path (dynamically obtained)
	apiPrefix := config.GlobalConfig.Server.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	monitorPrefix := config.GlobalConfig.Server.MonitorPrefix
	if monitorPrefix == "" {
		monitorPrefix = "/metrics"
	}
	// Combine complete monitoring URL path: /api/metrics/ui
	// Add monitor_url to Site object, as frontend accesses site.Site.monitor_url
	if siteMap, ok := siteCtx["Site"].(map[string]any); ok {
		siteMap["monitor_url"] = apiPrefix + monitorPrefix + "/ui"
	}

	c.JSON(http.StatusOK, gin.H{
		"objects": viewObjects,
		"user":    CurrentUser(c),
		"site":    siteCtx,
	})
}

// RegisterAdmin registers admin routes
func (obj *AdminObject) RegisterAdmin(r gin.IRoutes) {
	r = r.Use(func(ctx *gin.Context) {
		if obj.AccessCheck != nil {
			err := obj.AccessCheck(ctx, obj)
			if err != nil {
				LingEcho.AbortWithJSONError(ctx, http.StatusForbidden, err)
				return
			}
		}
		ctx.Next()
	})

	r.POST("/", obj.handleQueryOrGetOne)
	r.PUT("/", obj.handleCreate)
	r.PATCH("/", obj.handleUpdate)
	r.DELETE("/", obj.handleDelete)
	r.POST("/:name", obj.handleAction)
}

func (obj *AdminObject) asColNames(db *gorm.DB, fields []string) []string {
	for i := 0; i < len(fields); i++ {
		fields[i] = db.NamingStrategy.ColumnName(obj.tableName, fields[i])
	}
	return fields
}

// Build fill the properties of obj.
func (obj *AdminObject) Build(db *gorm.DB) error {
	if obj.Path == "" {
		obj.Path = strings.ToLower(obj.Name)
	}

	if obj.Path == "_" || obj.Path == "" {
		return fmt.Errorf("invalid path")
	}

	rt := reflect.TypeOf(obj.Model)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	obj.modelElem = rt
	obj.tableName = db.NamingStrategy.TableName(rt.Name())
	obj.PluralName = inflection.Plural(obj.Name)
	obj.Shows = obj.asColNames(db, obj.Shows)
	obj.Editables = obj.asColNames(db, obj.Editables)
	obj.Orderables = obj.asColNames(db, obj.Orderables)
	obj.Searchables = obj.asColNames(db, obj.Searchables)
	obj.Filterables = obj.asColNames(db, obj.Filterables)
	obj.Requireds = obj.asColNames(db, obj.Requireds)
	obj.primaryKeyMaping = map[string]string{}

	for idx := range obj.Orders {
		o := &obj.Orders[idx]
		o.Name = db.NamingStrategy.ColumnName(obj.tableName, o.Name)
	}

	obj.ignores = map[string]bool{}
	err := obj.parseFields(db, rt)
	if err != nil {
		return err
	}
	if len(obj.PrimaryKeys) <= 0 && len(obj.UniqueKeys) <= 0 {
		return fmt.Errorf("%s not has primaryKey or uniqueKeys", obj.Name)
	}

	for idx := range obj.Actions {
		action := &obj.Actions[idx]
		if action.Name == "" {
			continue
		}
		if action.Path == "" {
			action.Path = strings.ToLower(action.Name)
		}
	}
	return nil
}

func (obj *AdminObject) parseFields(db *gorm.DB, rt reflect.Type) error {
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)

		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			obj.parseFields(db, f.Type)
			continue
		}

		if f.Type.Kind() == reflect.Chan || f.Type.Kind() == reflect.Func || !f.IsExported() {
			continue
		}

		gormTag := strings.ToLower(f.Tag.Get("gorm"))
		field := AdminField{
			Name:      db.NamingStrategy.ColumnName(obj.tableName, f.Name),
			Tag:       gormTag,
			elemType:  f.Type,
			fieldName: f.Name,
			Label:     f.Tag.Get("label"),
			NotColumn: gormTag == "-",
		}
		if field.elemType.Kind() == reflect.Ptr {
			field.elemType = field.elemType.Elem()
		}
		if field.Label == "" {
			field.Label = strings.ReplaceAll(field.Name, "_", " ")
		}

		field.Label = cases.Title(language.Und).String(field.Label)

		switch f.Type.Kind() {
		case reflect.Ptr:
			field.Type = f.Type.Elem().Name()
			field.CanNull = true
			field.IsPtr = true
		case reflect.Slice:
			field.Type = f.Type.Elem().Name()
			field.CanNull = true
			field.IsArray = true
		default:
			field.Type = f.Type.Name()
		}

		if strings.Contains(gormTag, "primarykey") {
			field.Primary = true
			//obj.primaryKeys = append(obj.primaryKeys, field.Name)
			if strings.Contains(field.Type, "int") {
				field.IsAutoID = true
			}
		}

		if strings.Contains(gormTag, "primarykey") || strings.Contains(gormTag, "unique") {
			// hint foreignField
			keyName := field.Name
			if strings.HasSuffix(f.Name, "ID") {
				n := f.Name[:len(f.Name)-2]
				if ff, ok := rt.FieldByName(n); ok {
					if ff.Type.Kind() == reflect.Struct || (ff.Type.Kind() == reflect.Ptr && ff.Type.Elem().Kind() == reflect.Struct) {
						keyName = db.NamingStrategy.ColumnName(obj.tableName, ff.Name)
					}
				}
				obj.primaryKeyMaping[keyName] = field.Name
			}
			if strings.Contains(gormTag, "primarykey") {
				obj.PrimaryKeys = append(obj.PrimaryKeys, keyName)
			} else {
				obj.UniqueKeys = append(obj.UniqueKeys, keyName)
			}
		}

		foreignKey := ""
		// ignore `belongs to` and `has one` relation
		if f.Type.Kind() == reflect.Struct || (f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct) {
			hintForeignKey := fmt.Sprintf("%sID", f.Name)
			if _, ok := rt.FieldByName(hintForeignKey); ok {
				foreignKey = hintForeignKey
			}
		}
		if strings.Contains(gormTag, "foreignkey") {
			//extract foreign key from gorm tag with regex
			//example: foreignkey:UserRefer
			var re = regexp.MustCompile(`foreignkey:([a-zA-Z0-9]+)`)
			matches := re.FindStringSubmatch(gormTag)
			if len(matches) > 1 {
				foreignKey = strings.TrimSpace(matches[1])
			}
		}

		if foreignKey != "" {
			obj.ignores[foreignKey] = true
			for k := range obj.Fields {
				if obj.Fields[k].fieldName == foreignKey {
					// remove foreign key from fields
					obj.Fields = append(obj.Fields[:k], obj.Fields[k+1:]...)
					break
				}
			}

			field.Foreign = &AdminForeign{
				Field:      db.NamingStrategy.ColumnName(obj.tableName, foreignKey),
				Path:       strings.ToLower(f.Type.Name()),
				foreignKey: foreignKey,
				fieldName:  f.Name,
			}
		}

		if field.Type == "NullTime" || field.Type == "Time" || field.Type == "DeletedAt" {
			field.Type = "datetime"
		}

		if field.Type == "DeletedAt" || strings.HasPrefix("Null", field.Type) {
			field.CanNull = true
		}

		if attr, ok := obj.Attributes[f.Name]; ok {
			field.Attribute = &attr
		}
		obj.Fields = append(obj.Fields, field)
	}
	return nil
}

func formatAsInt64(v any) int64 {
	srcKind := reflect.ValueOf(v).Kind()
	switch srcKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(v).Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(reflect.ValueOf(v).Uint())
	case reflect.Float32, reflect.Float64:
		return int64(reflect.ValueOf(v).Float())
	case reflect.String:
		if v.(string) == "" {
			return 0
		}
		if i, err := strconv.ParseInt(v.(string), 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func convertValue(elemType reflect.Type, source any, targetIsPtr bool) (any, error) {
	srcType := reflect.TypeOf(source)
	if srcType == elemType {
		return source, nil
	}
	var targetType = elemType
	var err error
	switch elemType.Name() {
	case "int", "int8", "int16", "int32", "int64":
		v := formatAsInt64(source)
		return reflect.ValueOf(v).Convert(targetType).Interface(), nil
	case "uint", "uint8", "uint16", "uint32", "uint64":
		v := formatAsInt64(source)
		return reflect.ValueOf(v).Convert(targetType).Interface(), nil
	case "float32", "float64":
		v, err := strconv.ParseFloat(fmt.Sprintf("%v", source), 64)
		if err != nil {
			return nil, err
		}
		return reflect.ValueOf(v).Convert(targetType).Interface(), nil
	case "bool":
		val := fmt.Sprintf("%v", source)
		if val == "on" {
			val = "true"
		} else if val == "off" {
			val = "false"
		} else if val == "" {
			val = "false"
		}

		v, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		return reflect.ValueOf(v).Interface(), nil
	case "string":
		return fmt.Sprintf("%v", source), nil
	case "NullTime":
		tv, ok := source.(string)
		if tv == "" || !ok {
			if targetIsPtr {
				return nil, nil
			}
			return &sql.NullTime{}, nil
		} else {
			for _, tf := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02", time.RFC1123} {
				t, err := time.Parse(tf, tv)
				if err == nil {
					return &sql.NullTime{Time: t, Valid: true}, nil
				}
			}
		}
		return nil, fmt.Errorf("invalid datetime format %v", source)
	case "Time":
		tv, ok := source.(string)
		if tv == "" || !ok {
			if targetIsPtr {
				return nil, nil
			}
			return &time.Time{}, nil
		} else {
			for _, tf := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02", time.RFC1123} {
				t, err := time.Parse(tf, tv)
				if err == nil {
					return &t, nil
				}
			}
		}
		return nil, fmt.Errorf("invalid datetime format %v", source)
	default:
		var data []byte
		data, err = json.Marshal(source)
		if err != nil {
			return nil, err
		}
		value := reflect.New(targetType).Interface()
		err = json.Unmarshal(data, value)
		return value, err
	}
}

func (obj *AdminObject) UnmarshalFrom(elemObj reflect.Value, keys, vals map[string]any) (any, error) {
	if len(obj.Editables) > 0 {
		editables := make(map[string]bool)
		for _, v := range obj.Editables {
			editables[v] = true
		}
		for k := range vals {
			if _, ok := editables[k]; !ok {
				delete(vals, k)
			}
		}
	}

	for k, v := range keys {
		// if primary key in editables, then ignore it
		if _, ok := vals[k]; !ok {
			vals[k] = v
		}
	}

	for _, field := range obj.Fields {
		val, ok := vals[field.Name]
		if !ok {
			continue
		}

		if val == nil {
			continue
		}
		var target reflect.Value
		var targetValue reflect.Value
		var targetType = field.elemType
		if field.Foreign != nil {
			target = elemObj.Elem().FieldByName(field.Foreign.foreignKey)
			targetType = target.Type()
			if valMap, ok := val.(map[string]any); ok {
				if v, ok := valMap["value"]; ok {
					val = v
				}
			}
		} else {
			target = elemObj.Elem().FieldByName(field.fieldName)
		}

		fieldValue, err := convertValue(targetType, val, field.IsPtr)
		if err != nil {
			return nil, fmt.Errorf("invalid type: %s except: %s actual: %s error:%v", field.Name, field.Type, reflect.TypeOf(val).Name(), err)
		}
		targetValue = reflect.ValueOf(fieldValue)

		if target.Kind() == reflect.Ptr {
			ptrValue := reflect.New(reflect.PointerTo(field.elemType))
			if fieldValue != nil {
				ptrValue.Elem().Set(targetValue)
			}
			targetValue = ptrValue.Elem()
		} else {
			if targetValue.Kind() == reflect.Ptr {
				targetValue = targetValue.Elem()
			}
		}
		target.Set(targetValue)
	}
	return elemObj.Interface(), nil
}

func (obj *AdminObject) MarshalOne(c *gin.Context, val interface{}) (map[string]any, error) {
	var result = make(map[string]any)
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	for _, field := range obj.Fields {
		var fieldVal any
		if field.Foreign != nil {
			v := AdminValue{
				Value: rv.FieldByName(field.Foreign.foreignKey).Interface(),
			}
			fv := rv.FieldByName(field.Foreign.fieldName)
			if fv.IsValid() {
				if sv, ok := fv.Interface().(fmt.Stringer); ok {
					v.Label = sv.String()
				}
			}
			if v.Label == "" {
				v.Label = fmt.Sprintf("%v", v.Value)
			}
			fieldVal = v
		} else {
			v := rv.FieldByName(field.fieldName)
			if v.IsValid() {
				fieldVal = v.Interface()
			}
		}
		result[field.Name] = fieldVal
	}

	if obj.ViewOnSite != nil {
		result["_adminExtra"] = map[string]any{
			"viewOnSite": obj.ViewOnSite(LingEcho.GetDbConnection(c, obj.GetDB, false), c, val),
		}
	}

	return result, nil
}

func (obj *AdminObject) getPrimaryValues(c *gin.Context) map[string]any {
	var result = make(map[string]any)
	hasPrimaryQuery := false
	for _, field := range obj.PrimaryKeys {
		if v := c.Query(field); v != "" {
			result[field] = v
			hasPrimaryQuery = true
		}
	}

	if hasPrimaryQuery {
		return result
	}

	for _, field := range obj.UniqueKeys {
		if key, ok := obj.primaryKeyMaping[field]; ok {
			field = key
		}
		if v := c.Query(field); v != "" {
			result[field] = v
		}
	}
	return result
}

func (obj *AdminObject) handleGetOne(c *gin.Context) {
	db := LingEcho.GetDbConnection(c, obj.GetDB, false)
	modelObj := reflect.New(obj.modelElem).Interface()
	keys := obj.getPrimaryValues(c)
	if len(keys) <= 0 {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid primary key"))
		return
	}

	result := db.Preload(clause.Associations).Where(keys).First(modelObj)

	if result.Error != nil {
		LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, result.Error)
		return
	}

	if obj.BeforeRender != nil {
		rr, err := obj.BeforeRender(db, c, modelObj)
		if err != nil {
			LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, err)
			return
		}
		if rr != nil {
			// if BeforeRender return not nil, then use it as result
			modelObj = rr
		}
	}

	data, err := obj.MarshalOne(c, modelObj)
	if err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (obj *AdminObject) QueryObjects(session *gorm.DB, form *LingEcho.QueryForm, ctx *gin.Context) (r AdminQueryResult, err error) {
	for _, v := range form.Filters {
		if q := v.GetQuery(); q != "" {
			if v.Op == LingEcho.FilterOpLike {
				if kws, ok := v.Value.([]any); ok {
					qs := []string{}
					for _, kw := range kws {
						k := fmt.Sprintf("\"%%%s%%\"", strings.ReplaceAll(kw.(string), "\"", "\\\""))
						q := fmt.Sprintf("`%s`.`%s` LIKE %s", obj.tableName, v.Name, k)
						qs = append(qs, q)
					}
					session = session.Where(strings.Join(qs, " OR "))
				} else {
					session = session.Where(fmt.Sprintf("`%s`.%s", obj.tableName, q), fmt.Sprintf("%%%s%%", v.Value))
				}
			} else if v.Op == LingEcho.FilterOpBetween {
				vt := reflect.ValueOf(v.Value)
				if vt.Kind() != reflect.Slice && vt.Len() != 2 {
					return r, fmt.Errorf("invalid between value, must be slice with 2 elements")
				}
				session = session.Where(fmt.Sprintf("`%s`.%s", obj.tableName, q), vt.Index(0).Interface(), vt.Index(1).Interface())
			} else {
				session = session.Where(fmt.Sprintf("`%s`.%s", obj.tableName, q), v.Value)
			}
		}
	}

	var orders []LingEcho.Order
	if len(form.Orders) > 0 {
		orders = form.Orders
	} else {
		orders = obj.Orders
	}

	for _, v := range orders {
		if q := v.GetQuery(); q != "" && v.Op != "" {
			session = session.Order(fmt.Sprintf("`%s`.%s", obj.tableName, q))
		}
	}

	if form.Keyword != "" && len(obj.Searchables) > 0 {
		var query []string
		for _, v := range obj.Searchables {
			query = append(query, fmt.Sprintf("`%s`.`%s` LIKE @keyword", obj.tableName, v))
		}
		searchKey := strings.Join(query, " OR ")
		session = session.Where(searchKey, sql.Named("keyword", "%"+form.Keyword+"%"))
	}

	r.Pos = form.Pos
	r.Limit = form.Limit
	r.Keyword = form.Keyword

	session = session.Model(obj.Model)

	var c int64
	if err := session.Count(&c).Error; err != nil {
		return r, err
	}
	if c <= 0 {
		return r, nil
	}
	r.TotalCount = int(c)

	selected := []string{}
	for _, v := range obj.Fields {
		if v.NotColumn {
			continue
		}
		if v.Foreign != nil {
			selected = append(selected, v.Foreign.Field)
		} else {
			selected = append(selected, v.Name)
		}
	}

	vals := reflect.New(reflect.SliceOf(obj.modelElem))
	tx := session.Preload(clause.Associations).Select(selected).Offset(form.Pos)
	if form.Limit > 0 {
		tx = tx.Limit(form.Limit)
	}
	result := tx.Find(vals.Interface())
	if result.Error != nil {
		return r, result.Error
	}

	for i := 0; i < vals.Elem().Len(); i++ {
		modelObj := vals.Elem().Index(i).Addr().Interface()
		r.objects = append(r.objects, modelObj)
		if obj.BeforeRender != nil {
			db := LingEcho.GetDbConnection(ctx, obj.GetDB, false)
			rr, err := obj.BeforeRender(db, ctx, modelObj)
			if err != nil {
				return r, err
			}
			if rr != nil {
				// if BeforeRender return not nil, then use it as result
				modelObj = rr
			}
		}
		item, err := obj.MarshalOne(ctx, modelObj)
		if err != nil {
			return r, err
		}
		r.Items = append(r.Items, item)
	}
	return r, nil
}

// Query many objects with filter/limit/offset/order/search
func (obj *AdminObject) handleQueryOrGetOne(c *gin.Context) {
	if c.Request.ContentLength <= 0 {
		obj.handleGetOne(c)
		return
	}

	db, form, err := LingEcho.DefaultPrepareQuery(LingEcho.GetDbConnection(c, obj.GetDB, false), c)
	if err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if form.ForeignMode {
		form.Limit = 0 // TODO: support foreign mode limit
	}

	r, err := obj.QueryObjects(db, form, c)

	if err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}
	if form.ForeignMode {
		var items []map[string]any
		for i := 0; i < len(r.Items); i++ {
			item := map[string]any{}
			var valueVal any
			for _, v := range obj.Fields {
				if v.Primary {
					valueVal = r.Items[i][v.Name]
				}
			}
			if valueVal == nil {
				continue
			}
			item["value"] = valueVal
			iv := r.objects[i]
			if sv, ok := iv.(fmt.Stringer); ok {
				item["label"] = sv.String()
			} else {
				item["label"] = fmt.Sprintf("%v", valueVal)
			}
			items = append(items, item)
		}
		r.Items = items
	}
	c.JSON(http.StatusOK, r)
}

func (obj *AdminObject) handleCreate(c *gin.Context) {
	keys := obj.getPrimaryValues(c)
	var vals map[string]any
	if err := c.BindJSON(&vals); err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	elmObj := reflect.New(obj.modelElem)
	elm, err := obj.UnmarshalFrom(elmObj, keys, vals)
	if err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	db := LingEcho.GetDbConnection(c, obj.GetDB, true)
	if obj.BeforeCreate != nil {
		if err := obj.BeforeCreate(db, c, elm); err != nil {
			LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	result := db.Create(elm)
	if result.Error != nil {
		LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, result.Error)
		return
	}
	if obj.BeforeRender != nil {
		rr, err := obj.BeforeRender(db, c, elm)
		if err != nil {
			LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, err)
			return
		}
		if rr != nil {
			// if BeforeRender return not nil, then use it as result
			elm = rr
		}
	}
	c.JSON(http.StatusOK, elm)
}

func (obj *AdminObject) handleUpdate(c *gin.Context) {
	keys := obj.getPrimaryValues(c)
	if len(keys) <= 0 {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid primary key"))
		return
	}

	var inputVals map[string]any
	if err := c.BindJSON(&inputVals); err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	db := LingEcho.GetDbConnection(c, obj.GetDB, false)
	elmObj := reflect.New(obj.modelElem)
	err := db.Where(keys).First(elmObj.Interface()).Error
	if err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusNotFound, errors.New("not found"))
		return
	}

	val, err := obj.UnmarshalFrom(elmObj, keys, inputVals)
	if err != nil {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	if obj.BeforeUpdate != nil {
		if err := obj.BeforeUpdate(db, c, val, inputVals); err != nil {
			LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	conflictKeys := []clause.Column{}
	if len(obj.PrimaryKeys) > 0 {
		for _, k := range obj.PrimaryKeys {
			conflictKeys = append(conflictKeys, clause.Column{Name: k})
		}
	} else {
		for _, k := range obj.UniqueKeys {
			conflictKeys = append(conflictKeys, clause.Column{Name: k})
		}
	}

	for idx := range conflictKeys {
		k := &conflictKeys[idx]
		if v, ok := obj.primaryKeyMaping[k.Name]; ok {
			k.Name = v
		}
	}

	result := db.Clauses(clause.OnConflict{
		Columns:   conflictKeys,
		UpdateAll: true,
	}).Where(keys).Create(val)

	if result.Error != nil {
		LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, result.Error)
		return
	}
	c.JSON(http.StatusOK, true)
}

func (obj *AdminObject) handleDelete(c *gin.Context) {
	keys := obj.getPrimaryValues(c)
	if len(keys) <= 0 {
		LingEcho.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid primary key"))
		return
	}
	db := LingEcho.GetDbConnection(c, obj.GetDB, false)
	val := reflect.New(obj.modelElem).Interface()
	r := db.Where(keys).Take(val)

	// for gorm delete hook, need to load model first.
	if r.Error != nil {
		if errors.Is(r.Error, gorm.ErrRecordNotFound) {
			LingEcho.AbortWithJSONError(c, http.StatusNotFound, errors.New("not found"))
		} else {
			LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, r.Error)
		}
		return
	}

	if obj.BeforeDelete != nil {
		if err := obj.BeforeDelete(db, c, val); err != nil {
			LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
			return
		}
	}

	r = db.Where(keys).Delete(val)
	if r.Error != nil {
		LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, r.Error)
		return
	}
	c.JSON(http.StatusOK, true)
}

func (obj *AdminObject) handleAction(c *gin.Context) {
	for _, action := range obj.Actions {
		if action.Path != c.Param("name") {
			continue
		}

		db := LingEcho.GetDbConnection(c, obj.GetDB, false)
		if action.WithoutObject {
			handled, r, err := action.Handler(db, c, nil)
			if err != nil {
				LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, err)
				return
			}
			if !handled {
				c.JSON(http.StatusOK, r)
			}
			return
		}

		if action.Batch {
			var keys []map[string]any
			if err := json.Unmarshal([]byte(c.Query("keys")), &keys); err != nil {
				LingEcho.AbortWithJSONError(c, http.StatusBadRequest, err)
				return
			}
			handled, r, err := action.Handler(db, c, keys)
			if err != nil {
				LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, err)
				return
			}
			if !handled {
				c.JSON(http.StatusOK, r)
			}
			return
		}

		keys := obj.getPrimaryValues(c)
		if len(keys) <= 0 {
			LingEcho.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid primary key"))
			return
		}
		modelObj := reflect.New(obj.modelElem).Interface()
		result := db.Where(keys).First(modelObj)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				LingEcho.AbortWithJSONError(c, http.StatusNotFound, errors.New("not found"))
			} else {
				LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, result.Error)
			}
			return
		}
		handled, r, err := action.Handler(db, c, modelObj)
		if err != nil {
			LingEcho.AbortWithJSONError(c, http.StatusInternalServerError, err)
			return
		}

		if !handled {
			c.JSON(http.StatusOK, r)
		}
		return
	}
	c.AbortWithStatus(http.StatusBadRequest)
}
