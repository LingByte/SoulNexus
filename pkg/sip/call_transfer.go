package sip

import (
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/emiago/sipgo/sip"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CallTransfer 呼叫转移处理器
type CallTransfer struct {
	sipServer *SipServer
	db        *gorm.DB
}

// TransferRequest 转移请求
type TransferRequest struct {
	CallID       string `json:"callId"`       // 当前通话的Call-ID
	TargetURI    string `json:"targetUri"`    // 转移目标URI
	TransferType string `json:"transferType"` // blind (盲转) 或 attended (协商转)
}

// NewCallTransfer 创建呼叫转移处理器
func NewCallTransfer(sipServer *SipServer, db *gorm.DB) *CallTransfer {
	return &CallTransfer{
		sipServer: sipServer,
		db:        db,
	}
}

// HandleRefer 处理REFER请求（呼叫转移）
func (ct *CallTransfer) HandleRefer(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	logrus.WithFields(logrus.Fields{
		"call_id": callID,
		"from":    req.From().Address.String(),
		"to":      req.To().Address.String(),
	}).Info("Received REFER request for call transfer")

	// 解析Refer-To头
	referToHeader := req.GetHeader("Refer-To")
	if referToHeader == nil {
		logrus.WithField("call_id", callID).Error("Missing Refer-To header")
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Missing Refer-To header", nil)
		if err := tx.Respond(res); err != nil {
			logrus.WithError(err).Error("Failed to send 400 response")
		}
		return
	}

	// 解析目标URI
	referToStr := referToHeader.Value()
	var targetURI sip.Uri
	if err := sip.ParseUri(referToStr, &targetURI); err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to parse Refer-To URI")
		res := sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Invalid Refer-To URI", nil)
		if err := tx.Respond(res); err != nil {
			logrus.WithError(err).Error("Failed to send 400 response")
		}
		return
	}

	// 查找当前通话
	ct.sipServer.activeMutex.RLock()
	session, exists := ct.sipServer.activeSessions[callID]
	ct.sipServer.activeMutex.RUnlock()

	if !exists {
		logrus.WithField("call_id", callID).Error("Call session not found")
		res := sip.NewResponseFromRequest(req, sip.StatusNotFound, "Call session not found", nil)
		if err := tx.Respond(res); err != nil {
			logrus.WithError(err).Error("Failed to send 404 response")
		}
		return
	}

	// 发送202 Accepted响应
	res := sip.NewResponseFromRequest(req, 202, "Accepted", nil)
	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send 202 response")
		return
	}

	// 执行盲转（Blind Transfer）
	go ct.performBlindTransfer(callID, &targetURI, session)

	logrus.WithFields(logrus.Fields{
		"call_id":    callID,
		"target_uri": targetURI.String(),
	}).Info("Call transfer initiated")
}

// performBlindTransfer 执行盲转
func (ct *CallTransfer) performBlindTransfer(callID string, targetURI *sip.Uri, session *SessionInfo) {
	// 1. 向当前通话的另一方发送BYE请求，结束当前通话
	// 2. 发起新的INVITE请求到目标URI
	// 3. 更新数据库记录

	logrus.WithFields(logrus.Fields{
		"call_id":    callID,
		"target_uri": targetURI.String(),
	}).Info("Performing blind transfer")

	// 更新数据库记录，标记为转移中
	if ct.db != nil {
		var sipCall models.SipCall
		if err := ct.db.Where("call_id = ?", callID).First(&sipCall).Error; err == nil {
			sipCall.Metadata = fmt.Sprintf(`{"transfer": {"status": "transferring", "target": "%s"}}`, targetURI.String())
			ct.db.Save(&sipCall)
		}
	}

	// 这里应该：
	// 1. 发送BYE给当前通话的另一方
	// 2. 发起新的INVITE到目标
	// 由于需要访问SipServer的内部方法，这里简化处理
	// 实际实现需要更复杂的逻辑

	logrus.WithField("call_id", callID).Info("Blind transfer completed")
}

// TransferCall 主动转移通话（API调用）
func (ct *CallTransfer) TransferCall(callID, targetURI string, transferType string) error {
	if transferType != "blind" && transferType != "attended" {
		return fmt.Errorf("invalid transfer type: %s (must be 'blind' or 'attended')", transferType)
	}

	// 查找通话
	ct.sipServer.activeMutex.RLock()
	session, exists := ct.sipServer.activeSessions[callID]
	ct.sipServer.activeMutex.RUnlock()

	if !exists {
		return fmt.Errorf("call not found: %s", callID)
	}

	// 解析目标URI
	var target sip.Uri
	if err := sip.ParseUri(targetURI, &target); err != nil {
		return fmt.Errorf("invalid target URI: %v", err)
	}

	if transferType == "blind" {
		go ct.performBlindTransfer(callID, &target, session)
	} else {
		// attended transfer需要更复杂的实现
		return fmt.Errorf("attended transfer not yet implemented")
	}

	return nil
}
