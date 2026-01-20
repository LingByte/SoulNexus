package handlers

import (
	"reflect"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// convertOutgoingSession 转换sip包的OutgoingSession到handler的OutgoingSession
func convertOutgoingSession(sessionInterface interface{}) *OutgoingSession {
	// 使用反射获取字段值
	v := reflect.ValueOf(sessionInterface)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	resp := &OutgoingSession{}

	// 获取字段值
	if field := v.FieldByName("RemoteRTPAddr"); field.IsValid() {
		resp.RemoteRTPAddr = field.String()
	}
	if field := v.FieldByName("CallID"); field.IsValid() {
		resp.CallID = field.String()
	}
	if field := v.FieldByName("TargetURI"); field.IsValid() {
		resp.TargetURI = field.String()
	}
	if field := v.FieldByName("Status"); field.IsValid() {
		resp.Status = field.String()
	}
	if field := v.FieldByName("Error"); field.IsValid() {
		resp.Error = field.String()
	}
	if field := v.FieldByName("StartTime"); field.IsValid() {
		if t, ok := field.Interface().(time.Time); ok {
			resp.StartTime = t.Format("2006-01-02T15:04:05Z07:00")
		}
	}
	if field := v.FieldByName("AnswerTime"); field.IsValid() {
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if t, ok := field.Elem().Interface().(time.Time); ok {
				answerTime := t.Format("2006-01-02T15:04:05Z07:00")
				resp.AnswerTime = &answerTime
			}
		}
	}
	if field := v.FieldByName("EndTime"); field.IsValid() {
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if t, ok := field.Elem().Interface().(time.Time); ok {
				endTime := t.Format("2006-01-02T15:04:05Z07:00")
				resp.EndTime = &endTime
			}
		}
	}

	return resp
}

// SipHandler SIP相关处理器
type SipHandler struct {
	db        *gorm.DB
	sipServer SipServerInterface // SIP服务器接口
}

// SipServerInterface SIP服务器接口，用于解耦
type SipServerInterface interface {
	MakeOutgoingCall(targetURI string) (string, error)
	GetOutgoingSession(callID string) (interface{}, bool) // 返回sip包的OutgoingSession
	CancelOutgoingCall(callID string) error
	HangupOutgoingCall(callID string) error // 挂断已接通的通话
}

// OutgoingSession 呼出会话信息（与sip包中的结构对应）
type OutgoingSession struct {
	RemoteRTPAddr string
	CallID        string
	TargetURI     string
	Status        string
	StartTime     string
	AnswerTime    *string
	EndTime       *string
	Error         string
}

// NewSipHandler 创建SIP处理器
func NewSipHandler(db *gorm.DB, sipServer SipServerInterface) *SipHandler {
	return &SipHandler{
		db:        db,
		sipServer: sipServer,
	}
}

// MakeOutgoingCallRequest 发起呼出请求
type MakeOutgoingCallRequest struct {
	TargetURI string `json:"targetUri" binding:"required"` // 目标URI，如: sip:user@192.168.1.100:5060
	UserID    *uint  `json:"userId,omitempty"`             // 关联用户ID（可选）
	GroupID   *uint  `json:"groupId,omitempty"`            // 关联组织ID（可选）
	Notes     string `json:"notes,omitempty"`              // 备注
}

// MakeOutgoingCallResponse 发起呼出响应
type MakeOutgoingCallResponse struct {
	CallID    string `json:"callId"`    // 通话ID
	Status    string `json:"status"`    // 状态
	TargetURI string `json:"targetUri"` // 目标URI
}

// MakeOutgoingCall 发起呼出呼叫
// @Summary 发起SIP呼出呼叫
// @Description 发起一个SIP呼出呼叫到指定的URI
// @Tags SIP
// @Accept json
// @Produce json
// @Param request body MakeOutgoingCallRequest true "呼出请求"
// @Success 200 {object} response.Response{data=MakeOutgoingCallResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/sip/calls/outgoing [post]
func (h *SipHandler) MakeOutgoingCall(c *gin.Context) {
	var req MakeOutgoingCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request: "+err.Error(), nil)
		return
	}

	// 验证targetURI格式
	if req.TargetURI == "" {
		response.Fail(c, "targetUri is required", nil)
		return
	}

	// 检查SIP服务器是否可用
	if h.sipServer == nil {
		response.Fail(c, "SIP server is not available", nil)
		return
	}

	// 发起呼出
	callID, err := h.sipServer.MakeOutgoingCall(req.TargetURI)
	if err != nil {
		logrus.WithError(err).Error("Failed to make outgoing call")
		response.Fail(c, "Failed to make call: "+err.Error(), nil)
		return
	}

	// 创建通话记录
	sipCall := &models.SipCall{
		CallID:    callID,
		Direction: models.SipCallDirectionOutbound,
		Status:    models.SipCallStatusCalling,
		ToURI:     req.TargetURI,
		StartTime: time.Now(),
		UserID:    req.UserID,
		GroupID:   req.GroupID,
		Notes:     req.Notes,
	}

	if err := models.CreateSipCall(h.db, sipCall); err != nil {
		logrus.WithError(err).Warn("Failed to create call record")
		// 不返回错误，因为呼叫已经发起
	}

	response.Success(c, "Call initiated successfully", MakeOutgoingCallResponse{
		CallID:    callID,
		Status:    "calling",
		TargetURI: req.TargetURI,
	})
}

// GetOutgoingCallStatus 获取呼出状态
// @Summary 获取呼出呼叫状态
// @Description 根据CallID获取呼出呼叫的当前状态
// @Tags SIP
// @Produce json
// @Param callId path string true "通话ID"
// @Success 200 {object} response.Response{data=OutgoingSession}
// @Failure 404 {object} response.Response
// @Router /api/sip/calls/outgoing/{callId} [get]
func (h *SipHandler) GetOutgoingCallStatus(c *gin.Context) {
	callID := c.Param("callId")
	if callID == "" {
		response.Fail(c, "callId is required", nil)
		return
	}

	// 检查SIP服务器是否可用
	if h.sipServer == nil {
		response.Fail(c, "SIP server is not available", nil)
		return
	}

	// 先从数据库查询通话记录
	sipCall, err := models.GetSipCallByCallID(h.db, callID)
	if err != nil {
		// 如果数据库中没有，尝试从SIP服务器获取会话信息
		sessionInterface, exists := h.sipServer.GetOutgoingSession(callID)
		if !exists {
			response.Fail(c, "Call not found", nil)
			return
		}

		// 转换为响应格式
		resp := convertOutgoingSession(sessionInterface)
		if resp == nil {
			response.Fail(c, "Failed to convert session", nil)
			return
		}

		response.Success(c, "Success", resp)
		return
	}

	// 如果数据库中有记录，也尝试从SIP服务器获取最新状态
	var resp *OutgoingSession
	if sessionInterface, exists := h.sipServer.GetOutgoingSession(callID); exists {
		resp = convertOutgoingSession(sessionInterface)
	}

	// 如果SIP服务器中没有会话，但从数据库中有记录，返回数据库中的信息
	if resp == nil {
		// 从数据库记录构建响应
		startTime := sipCall.StartTime.Format("2006-01-02T15:04:05Z07:00")
		resp = &OutgoingSession{
			CallID:        sipCall.CallID,
			TargetURI:     sipCall.ToURI,
			Status:        string(sipCall.Status),
			StartTime:     startTime,
			RemoteRTPAddr: sipCall.RemoteRTPAddr,
			Error:         sipCall.ErrorMessage,
		}

		if sipCall.AnswerTime != nil {
			answerTime := sipCall.AnswerTime.Format("2006-01-02T15:04:05Z07:00")
			resp.AnswerTime = &answerTime
		}

		if sipCall.EndTime != nil {
			endTime := sipCall.EndTime.Format("2006-01-02T15:04:05Z07:00")
			resp.EndTime = &endTime
		}
	}

	response.Success(c, "Success", resp)
}

// CancelOutgoingCall 取消呼出呼叫
// @Summary 取消呼出呼叫
// @Description 取消一个正在进行的呼出呼叫
// @Tags SIP
// @Produce json
// @Param callId path string true "通话ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/sip/calls/outgoing/{callId}/cancel [post]
func (h *SipHandler) CancelOutgoingCall(c *gin.Context) {
	callID := c.Param("callId")
	if callID == "" {
		response.Fail(c, "callId is required", nil)
		return
	}

	// 检查SIP服务器是否可用
	if h.sipServer == nil {
		response.Fail(c, "SIP server is not available", nil)
		return
	}

	err := h.sipServer.CancelOutgoingCall(callID)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	// 更新通话记录状态
	sipCall, err := models.GetSipCallByCallID(h.db, callID)
	if err == nil {
		now := time.Now()
		sipCall.Status = models.SipCallStatusCancelled
		sipCall.EndTime = &now
		if err := models.UpdateSipCall(h.db, sipCall); err != nil {
			logrus.WithError(err).Warn("Failed to update call record")
		}
	}

	response.Success(c, "Call cancelled successfully", nil)
}

// HangupOutgoingCall 挂断呼出呼叫
// @Summary 挂断呼出呼叫
// @Description 挂断一个已接通的呼出呼叫
// @Tags SIP
// @Produce json
// @Param callId path string true "通话ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/sip/calls/outgoing/{callId}/hangup [post]
func (h *SipHandler) HangupOutgoingCall(c *gin.Context) {
	if h.sipServer == nil {
		response.Fail(c, "SIP server is not initialized", nil)
		return
	}

	callID := c.Param("callId")
	if callID == "" {
		response.Fail(c, "callId is required", nil)
		return
	}

	err := h.sipServer.HangupOutgoingCall(callID)
	if err != nil {
		if err.Error() == "call not found" {
			response.Fail(c, "Call not found", err.Error())
		} else {
			response.Fail(c, "Failed to hangup call", err.Error())
		}
		return
	}

	// 更新通话记录状态
	sipCall, err := models.GetSipCallByCallID(h.db, callID)
	if err == nil {
		now := time.Now()
		sipCall.Status = models.SipCallStatusEnded
		sipCall.EndTime = &now
		if sipCall.AnswerTime != nil {
			duration := int(now.Sub(*sipCall.AnswerTime).Seconds())
			if duration > 0 {
				sipCall.Duration = duration
			}
		}
		if err := models.UpdateSipCall(h.db, sipCall); err != nil {
			logrus.WithError(err).Warn("Failed to update call record status to ended")
		}
	} else {
		logrus.WithError(err).Warn("Call record not found for hangup update")
	}

	response.Success(c, "Call hung up successfully", gin.H{"message": "Call hung up successfully"})
}

// GetCallHistory 获取通话历史
// @Summary 获取通话历史
// @Description 获取通话记录列表
// @Tags SIP
// @Produce json
// @Param userId query int false "用户ID"
// @Param status query string false "状态筛选"
// @Param limit query int false "限制数量" default(20)
// @Success 200 {object} response.Response{data=[]models.SipCall}
// @Router /api/sip/calls [get]
func (h *SipHandler) GetCallHistory(c *gin.Context) {
	userIDStr := c.Query("userId")
	status := c.Query("status")
	limitStr := c.DefaultQuery("limit", "20")

	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	var calls []models.SipCall
	query := h.db.Order("created_at DESC")

	if userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			query = query.Where("user_id = ?", uint(userID))
		}
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Limit(limit).Find(&calls).Error; err != nil {
		logrus.WithError(err).Error("Failed to get call history")
		response.Fail(c, "Failed to get call history: "+err.Error(), nil)
		return
	}

	response.Success(c, "Success", calls)
}

// GetSipUsers 获取SIP用户列表
// @Summary 获取SIP用户列表
// @Description 获取所有SIP用户列表
// @Tags SIP
// @Produce json
// @Success 200 {object} response.Response{data=[]models.SipUser}
// @Router /api/sip/users [get]
func (h *SipHandler) GetSipUsers(c *gin.Context) {
	var sipUsers []models.SipUser
	query := h.db.Order("created_at DESC")

	// 可选：根据状态筛选
	status := c.Query("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 可选：只获取已启用的用户
	enabled := c.Query("enabled")
	if enabled == "true" {
		query = query.Where("enabled = ?", true)
	}

	if err := query.Find(&sipUsers).Error; err != nil {
		logrus.WithError(err).Error("Failed to get SIP users")
		response.Fail(c, "Failed to get SIP users: "+err.Error(), nil)
		return
	}

	response.Success(c, "Success", sipUsers)
}
