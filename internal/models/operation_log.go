package models

import (
	"strings"
	"time"

	constants2 "github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OperationLog is an append-only audit row for console write operations.
type OperationLog struct {
	common.BaseModel

	TenantID     uint   `json:"tenantId,string" gorm:"index;not null;default:0;comment:租户ID，0表示平台级"`
	Operator     string `json:"operator" gorm:"size:128;index;comment:操作人标识"`
	OperatorKind string `json:"operatorKind" gorm:"size:32;not null;default:'';index;comment:操作人类型"`
	OperatorID   uint   `json:"operatorId,string" gorm:"not null;default:0;comment:操作人ID"`
	Action       string `json:"action" gorm:"size:32;not null;index;comment:动作"`
	Resource     string `json:"resource" gorm:"size:64;not null;index;comment:资源类型"`
	ResourceID   uint   `json:"resourceId,string" gorm:"not null;default:0;index;comment:资源ID"`
	ResourceName string `json:"resourceName,omitempty" gorm:"size:256;comment:资源名称"`
	Summary      string `json:"summary,omitempty" gorm:"size:512;comment:摘要"`
	DetailJSON   string `json:"detailJson,omitempty" gorm:"column:detail_json;type:text;comment:详情JSON"`
	HTTPMethod   string `json:"httpMethod" gorm:"size:16;comment:HTTP方法"`
	HTTPPath     string `json:"httpPath" gorm:"size:512;comment:HTTP路径"`
	RequestID    string `json:"requestId" gorm:"size:64;index;comment:请求ID"`
	ClientIP     string `json:"clientIP" gorm:"column:client_ip;size:64;comment:客户端IP"`
	Success      bool   `json:"success" gorm:"not null;default:true;index;comment:是否成功"`
	ErrorMsg     string `json:"errorMsg,omitempty" gorm:"size:512;comment:错误信息"`
}

func (OperationLog) TableName() string {
	return constants.OPERATION_LOG_TABLE_NAME
}

// OperationLogListFilter scopes list queries.
type OperationLogListFilter struct {
	TenantID      uint
	Operator      string
	OperatorExact string
	Action        string
	Resource      string
	ResourceID    uint
	Success       *bool
	From          *time.Time
	To            *time.Time
}

// OperationLogMineView is the tenant-user scoped audit row (no tenantId).
type OperationLogMineView struct {
	ID           uint      `json:"id,string"`
	CreatedAt    time.Time `json:"createdAt"`
	Operator     string    `json:"operator"`
	OperatorKind string    `json:"operatorKind"`
	OperatorID   uint      `json:"operatorId,string"`
	Action       string    `json:"action"`
	Resource     string    `json:"resource"`
	ResourceID   uint      `json:"resourceId,string"`
	ResourceName string    `json:"resourceName,omitempty"`
	Summary      string    `json:"summary,omitempty"`
	DetailJSON   string    `json:"detailJson,omitempty"`
	HTTPMethod   string    `json:"httpMethod"`
	HTTPPath     string    `json:"httpPath"`
	RequestID    string    `json:"requestId"`
	ClientIP     string    `json:"clientIP"`
	Success      bool      `json:"success"`
	ErrorMsg     string    `json:"errorMsg,omitempty"`
}

func ToOperationLogMineView(row OperationLog) OperationLogMineView {
	return OperationLogMineView{
		ID:           row.ID,
		CreatedAt:    row.CreatedAt,
		Operator:     row.Operator,
		OperatorKind: row.OperatorKind,
		OperatorID:   row.OperatorID,
		Action:       row.Action,
		Resource:     row.Resource,
		ResourceID:   row.ResourceID,
		ResourceName: row.ResourceName,
		Summary:      row.Summary,
		DetailJSON:   row.DetailJSON,
		HTTPMethod:   row.HTTPMethod,
		HTTPPath:     row.HTTPPath,
		RequestID:    row.RequestID,
		ClientIP:     row.ClientIP,
		Success:      row.Success,
		ErrorMsg:     row.ErrorMsg,
	}
}

// ListOperationLogsPage returns paginated operation logs newest first.
func ListOperationLogsPage(db *gorm.DB, page, size int, f OperationLogListFilter) ([]OperationLog, int64, error) {
	if db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	q := db.Model(&OperationLog{})
	if f.TenantID > 0 {
		q = q.Where("tenant_id = ?", f.TenantID)
	}
	if s := strings.TrimSpace(f.OperatorExact); s != "" {
		q = q.Where("operator = ?", s)
	} else if s := strings.TrimSpace(f.Operator); s != "" {
		q = q.Where("operator LIKE ?", "%"+s+"%")
	}
	if s := strings.TrimSpace(f.Action); s != "" {
		q = q.Where("action = ?", s)
	}
	if s := strings.TrimSpace(f.Resource); s != "" {
		q = q.Where("resource = ?", s)
	}
	if f.ResourceID > 0 {
		q = q.Where("resource_id = ?", f.ResourceID)
	}
	if f.Success != nil {
		q = q.Where("success = ?", *f.Success)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	return utils.FindPageQuery[OperationLog](q, page, size, utils.MaxPageSize200, func(q *gorm.DB) *gorm.DB {
		return q.Order("created_at DESC").Order("id DESC")
	})
}

// InsertOperationLog persists one audit row.
func InsertOperationLog(db *gorm.DB, row *OperationLog) error {
	if db == nil || row == nil {
		return gorm.ErrInvalidDB
	}
	return db.Create(row).Error
}

// OperationLogEvent is emitted by models/tasks when data changes outside HTTP handlers.
type OperationLogEvent struct {
	TenantID     uint
	Operator     string
	OperatorKind string
	OperatorID   uint
	Action       string
	Resource     string
	ResourceID   uint
	ResourceName string
	Summary      string
	Before       any
	After        any
	Request      any
	Source       string
	Success      bool
	ErrorMsg     string
}

// EmitOperationLog persists a non-HTTP audit row.
func EmitOperationLog(db *gorm.DB, e OperationLogEvent) {
	if db == nil {
		return
	}
	WriteOperationLogFromEvent(db, e)
}

// OperationLogInput is the data layer input for one append-only audit row.
type OperationLogInput struct {
	TenantID     uint
	OperatorKind string
	OperatorID   uint
	Operator     string
	Action       string
	Resource     string
	ResourceID   uint
	ResourceName string
	Summary      string
	Before       any
	After        any
	Request      any
	Source       string
	Success      bool
	ErrorMsg     string
	HTTPMethod   string
	HTTPPath     string
	RequestID    string
	ClientIP     string
}

// WriteOperationLog persists one operation log row.
func WriteOperationLog(db *gorm.DB, in OperationLogInput) {
	if db == nil {
		return
	}
	if in.OperatorKind == "" {
		in.OperatorKind = constants2.OpOperatorSystem
	}
	if strings.TrimSpace(in.Operator) == "" {
		in.Operator = "system"
	}
	success := in.Success
	if in.ErrorMsg != "" {
		success = false
	}
	detail := audit.BuildDetailJSON(in.Before, in.After, in.Request, in.Source)
	row := &OperationLog{
		TenantID:     in.TenantID,
		Operator:     strings.TrimSpace(in.Operator),
		OperatorKind: strings.TrimSpace(in.OperatorKind),
		OperatorID:   in.OperatorID,
		Action:       strings.TrimSpace(in.Action),
		Resource:     strings.TrimSpace(in.Resource),
		ResourceID:   in.ResourceID,
		ResourceName: strings.TrimSpace(in.ResourceName),
		Summary:      strings.TrimSpace(in.Summary),
		DetailJSON:   audit.MarshalDetailJSON(detail),
		HTTPMethod:   strings.TrimSpace(in.HTTPMethod),
		HTTPPath:     strings.TrimSpace(in.HTTPPath),
		RequestID:    strings.TrimSpace(in.RequestID),
		ClientIP:     strings.TrimSpace(in.ClientIP),
		Success:      success,
		ErrorMsg:     strings.TrimSpace(in.ErrorMsg),
	}
	if err := InsertOperationLog(db, row); err != nil && logger.Lg != nil {
		logger.Lg.Warn("operation log insert failed",
			zap.Error(err),
			zap.String("action", row.Action),
			zap.String("resource", row.Resource),
			zap.Uint("resourceId", row.ResourceID),
		)
	}
}

// WriteOperationLogFromEvent maps a model/task-layer event to persistence.
func WriteOperationLogFromEvent(db *gorm.DB, e OperationLogEvent) {
	WriteOperationLog(db, OperationLogInput{
		TenantID:     e.TenantID,
		Operator:     e.Operator,
		OperatorKind: e.OperatorKind,
		OperatorID:   e.OperatorID,
		Action:       e.Action,
		Resource:     e.Resource,
		ResourceID:   e.ResourceID,
		ResourceName: e.ResourceName,
		Summary:      e.Summary,
		Before:       e.Before,
		After:        e.After,
		Request:      e.Request,
		Source:       e.Source,
		Success:      e.Success,
		ErrorMsg:     e.ErrorMsg,
	})
}
