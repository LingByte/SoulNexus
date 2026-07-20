package task

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// ExecutionTask is a durable background job row (params JSON + handler resolved at runtime).
type ExecutionTask struct {
	common.BaseModel

	TaskID     string     `json:"taskId" gorm:"size:64;uniqueIndex;not null;comment:调度任务ID"`
	QueueName  string     `json:"queueName" gorm:"size:64;index:idx_exec_task_queue_status,priority:1;not null;comment:队列名称"`
	Title      string     `json:"title,omitempty" gorm:"size:256;comment:任务标题"`
	Kind       string     `json:"kind,omitempty" gorm:"size:64;index;comment:任务类型"`
	Source     string     `json:"source,omitempty" gorm:"size:64;index;comment:任务来源模块"`
	TenantID   uint       `json:"tenantId,string,omitempty" gorm:"index;default:0;comment:关联租户ID"`
	Priority   int        `json:"priority" gorm:"not null;default:0;comment:优先级"`
	ParamsJSON string     `json:"paramsJson" gorm:"type:longtext;not null;comment:任务参数JSON"`
	ResultJSON string     `json:"resultJson,omitempty" gorm:"type:longtext;comment:执行结果JSON"`
	Status     string     `json:"status" gorm:"size:20;index:idx_exec_task_queue_status,priority:2;not null;comment:任务状态"`
	Progress   int        `json:"progress" gorm:"not null;default:0;comment:进度0-100"`
	RetryCount int        `json:"retryCount" gorm:"not null;default:0;comment:已重试次数"`
	MaxRetries int        `json:"maxRetries" gorm:"not null;default:3;comment:最大重试次数"`
	ErrorMsg   string     `json:"errorMsg,omitempty" gorm:"size:1024;comment:错误信息"`
	SubmitTime time.Time  `json:"submitTime" gorm:"index;not null;comment:提交时间"`
	StartedAt  *time.Time `json:"startedAt,omitempty" gorm:"comment:开始执行时间"`
	FinishedAt *time.Time `json:"finishedAt,omitempty" gorm:"comment:完成时间"`
	WorkerID   string     `json:"workerId,omitempty" gorm:"size:128;comment:执行节点标识"`
}

func (ExecutionTask) TableName() string { return constants.EXECUTION_TASK_TABLE_NAME }

// ExecutionTaskListFilter scopes admin list queries.
type ExecutionTaskListFilter struct {
	QueueName string
	Kind      string
	Source    string
	Status    string
	TenantID  uint
	TaskID    string
	Search    string
	From      *time.Time
	To        *time.Time
}

// ExecutionTaskStatsSummary aggregates task counts by status.
type ExecutionTaskStatsSummary struct {
	Total    int64            `json:"total"`
	ByQueue  map[string]int64 `json:"byQueue,omitempty"`
	ByStatus map[string]int64 `json:"byStatus"`
}

func workerNodeID() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	return host
}

// SaveExecutionTaskPending inserts or updates a pending task row keyed by TaskID.
func SaveExecutionTaskPending(db *gorm.DB, rec ExecutionTask) error {
	if db == nil {
		return nil
	}
	if rec.TaskID == "" || rec.QueueName == "" {
		return errors.New("task id and queue name are required")
	}
	rec.Status = TaskStatusPending.String()
	if rec.SubmitTime.IsZero() {
		rec.SubmitTime = time.Now()
	}
	if rec.MaxRetries <= 0 {
		rec.MaxRetries = 3
	}
	var existing ExecutionTask
	err := db.Where("task_id = ?", rec.TaskID).First(&existing).Error
	if err == nil {
		rec.ID = existing.ID
		rec.CreatedAt = existing.CreatedAt
		rec.CreateBy = existing.CreateBy
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Save(&rec).Error
}

// MarkExecutionTaskRunning marks a task as executing.
func MarkExecutionTaskRunning(db *gorm.DB, taskID string) error {
	if db == nil || taskID == "" {
		return nil
	}
	now := time.Now()
	return db.Model(&ExecutionTask{}).Where("task_id = ?", taskID).Updates(map[string]any{
		"status":     TaskStatusRunning.String(),
		"started_at": now,
		"progress":   1,
		"error_msg":  "",
		"worker_id":  workerNodeID(),
	}).Error
}

// MarkExecutionTaskFinished records terminal task state.
func MarkExecutionTaskFinished(db *gorm.DB, taskID string, status TaskStatus, errMsg string) error {
	if db == nil || taskID == "" {
		return nil
	}
	if errMsg != "" && len(errMsg) > 1000 {
		errMsg = errMsg[:1000]
	}
	now := time.Now()
	progress := 0
	if status == TaskStatusSuccess {
		progress = 100
	}
	return db.Model(&ExecutionTask{}).Where("task_id = ?", taskID).Updates(map[string]any{
		"status":      status.String(),
		"error_msg":   errMsg,
		"finished_at": now,
		"progress":    progress,
	}).Error
}

// ResetRunningExecutionTasksToPending re-queues tasks interrupted by a crash or kill.
func ResetRunningExecutionTasksToPending(db *gorm.DB, queue string) (int64, error) {
	if db == nil || queue == "" {
		return 0, nil
	}
	res := db.Model(&ExecutionTask{}).
		Where("queue_name = ? AND status = ?", queue, TaskStatusRunning.String()).
		Updates(map[string]any{
			"status":      TaskStatusPending.String(),
			"started_at":  nil,
			"finished_at": nil,
			"error_msg":   "",
			"progress":    0,
			"worker_id":   "",
		})
	return res.RowsAffected, res.Error
}

// ListPendingExecutionTasks returns pending tasks in dispatch order.
func ListPendingExecutionTasks(db *gorm.DB, queue string) ([]ExecutionTask, error) {
	if db == nil || queue == "" {
		return nil, nil
	}
	var rows []ExecutionTask
	err := db.Where("queue_name = ? AND status = ?", queue, TaskStatusPending.String()).
		Order("priority DESC, submit_time ASC").
		Find(&rows).Error
	return rows, err
}

// GetExecutionTaskByID loads one row by snowflake id.
func GetExecutionTaskByID(db *gorm.DB, id uint) (*ExecutionTask, error) {
	if db == nil || id == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row ExecutionTask
	err := db.First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetExecutionTaskByTaskID loads one row by scheduler task id.
func GetExecutionTaskByTaskID(db *gorm.DB, taskID string) (*ExecutionTask, error) {
	if db == nil || strings.TrimSpace(taskID) == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row ExecutionTask
	err := db.Where("task_id = ?", taskID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListExecutionTasksPage returns paginated admin rows.
func ListExecutionTasksPage(db *gorm.DB, filter ExecutionTaskListFilter, page, pageSize int) ([]ExecutionTask, int64, error) {
	if db == nil {
		return nil, 0, nil
	}
	q := db.Model(&ExecutionTask{})
	if v := strings.TrimSpace(filter.QueueName); v != "" {
		q = q.Where("queue_name = ?", v)
	}
	if v := strings.TrimSpace(filter.Kind); v != "" {
		q = q.Where("kind = ?", v)
	}
	if v := strings.TrimSpace(filter.Source); v != "" {
		q = q.Where("source = ?", v)
	}
	if v := strings.TrimSpace(filter.Status); v != "" && !strings.EqualFold(v, "all") {
		q = q.Where("status = ?", v)
	}
	if filter.TenantID > 0 {
		q = q.Where("tenant_id = ?", filter.TenantID)
	}
	if v := strings.TrimSpace(filter.TaskID); v != "" {
		q = q.Where("task_id = ?", v)
	}
	if v := strings.TrimSpace(filter.Search); v != "" {
		like := "%" + v + "%"
		q = q.Where("title LIKE ? OR task_id LIKE ? OR queue_name LIKE ?", like, like, like)
	}
	if filter.From != nil {
		q = q.Where("submit_time >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("submit_time <= ?", *filter.To)
	}
	return utils.FindPage[ExecutionTask](q, page, pageSize, "submit_time DESC, id DESC", utils.MaxPageSize200)
}

// ExecutionTaskStats returns aggregate counts for admin dashboards.
func ExecutionTaskStats(db *gorm.DB, queueName string) (*ExecutionTaskStatsSummary, error) {
	if db == nil {
		return &ExecutionTaskStatsSummary{ByStatus: map[string]int64{}}, nil
	}
	type statusRow struct {
		Status string
		Cnt    int64
	}
	q := db.Model(&ExecutionTask{})
	if v := strings.TrimSpace(queueName); v != "" {
		q = q.Where("queue_name = ?", v)
	}
	var rows []statusRow
	if err := q.Select("status, COUNT(*) AS cnt").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := &ExecutionTaskStatsSummary{ByStatus: make(map[string]int64, len(rows))}
	for _, r := range rows {
		out.ByStatus[r.Status] = r.Cnt
		out.Total += r.Cnt
	}
	return out, nil
}

// UpdateExecutionTask applies admin edits to a row.
func UpdateExecutionTask(db *gorm.DB, id uint, updates map[string]any) error {
	if db == nil || id == 0 {
		return gorm.ErrRecordNotFound
	}
	if len(updates) == 0 {
		return nil
	}
	res := db.Model(&ExecutionTask{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SoftDeleteExecutionTask soft-deletes a task row.
func SoftDeleteExecutionTask(db *gorm.DB, id uint, operator string) error {
	if db == nil || id == 0 {
		return gorm.ErrRecordNotFound
	}
	var row ExecutionTask
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	row.SoftDelete(operator)
	return db.Save(&row).Error
}

// ResetExecutionTaskForRetry marks a failed/canceled task pending again.
func ResetExecutionTaskForRetry(db *gorm.DB, taskID string) error {
	if db == nil || strings.TrimSpace(taskID) == "" {
		return gorm.ErrRecordNotFound
	}
	res := db.Model(&ExecutionTask{}).Where("task_id = ?", taskID).Updates(map[string]any{
		"status":      TaskStatusPending.String(),
		"error_msg":   "",
		"started_at":  nil,
		"finished_at": nil,
		"progress":    0,
		"worker_id":   "",
		"retry_count": gorm.Expr("retry_count + 1"),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
