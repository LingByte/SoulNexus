package ua

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RegistrationInfo contains extracted registration information from SIP REGISTER request
type RegistrationInfo struct {
	Username    string
	ContactStr  string
	ContactIP   string
	ContactPort int
	Expires     int
	UserAgent   string
	RemoteIP    string
}

// SavePendingSession saves a pending session based on storage type
func (c *UAConfig) SavePendingSession(callID, clientRTPAddr string) error {
	switch c.StorageType {
	case StorageTypeDatabase:
		return c.savePendingSessionToDatabase(callID, clientRTPAddr)

	case StorageTypeFile:
		return c.savePendingSessionToFile(callID, clientRTPAddr)

	case StorageTypeMemory:
		return c.savePendingSessionToMemory(callID, clientRTPAddr)

	default:
		// Default to memory
		return c.savePendingSessionToMemory(callID, clientRTPAddr)
	}
}

// GetPendingSession gets a pending session based on storage type
func (c *UAConfig) GetPendingSession(callID string) (string, bool) {
	switch c.StorageType {
	case StorageTypeDatabase:
		return c.getPendingSessionFromDatabase(callID)

	case StorageTypeFile:
		return c.getPendingSessionFromFile(callID)

	case StorageTypeMemory:
		return c.getPendingSessionFromMemory(callID)

	default:
		// Default to memory
		return c.getPendingSessionFromMemory(callID)
	}
}

// RemovePendingSession removes a pending session based on storage type
func (c *UAConfig) RemovePendingSession(callID string) error {
	switch c.StorageType {
	case StorageTypeDatabase:
		return c.removePendingSessionFromDatabase(callID)

	case StorageTypeFile:
		return c.removePendingSessionFromFile(callID)

	case StorageTypeMemory:
		return c.removePendingSessionFromMemory(callID)

	default:
		// Default to memory
		return c.removePendingSessionFromMemory(callID)
	}
}

// ==================== Call Storage ====================

// SaveCall saves a call record based on storage type
func (c *UAConfig) SaveCall(sipCall *models.SipCall) error {
	switch c.StorageType {
	case StorageTypeDatabase:
		return c.saveCallToDatabase(sipCall)

	case StorageTypeFile:
		// File storage for calls should be handled by saveInviteToFile in func_register.go
		return fmt.Errorf("file storage for calls should be handled by saveInviteToFile")

	case StorageTypeMemory:
		return c.saveCallToMemory(sipCall)

	default:
		// Default to memory
		return c.saveCallToMemory(sipCall)
	}
}

// GetCall gets a call record based on storage type
func (c *UAConfig) GetCall(callID string) (*models.SipCall, bool) {
	switch c.StorageType {
	case StorageTypeDatabase:
		return c.getCallFromDatabase(callID)

	case StorageTypeFile:
		// File storage retrieval not implemented
		return nil, false

	case StorageTypeMemory:
		return c.getCallFromMemory(callID)

	default:
		// Default to memory
		return c.getCallFromMemory(callID)
	}
}

// UpdateCallStatusInMemory updates call status in memory storage
func (c *UAConfig) UpdateCallStatusInMemory(callID string, status models.SipCallStatus, answerTime *time.Time) {
	c.memoryCallsMutex.Lock()
	defer c.memoryCallsMutex.Unlock()
	if call, exists := c.memoryCalls[callID]; exists {
		call.Status = status
		if answerTime != nil {
			call.AnswerTime = answerTime
		}
	}
}

// ==================== Active Session Storage ====================

// SaveActiveSession saves an active session
func (c *UAConfig) SaveActiveSession(callID string, session *SessionInfo) {
	c.activeMutex.Lock()
	defer c.activeMutex.Unlock()
	c.activeSessions[callID] = session
}

// GetActiveSession gets an active session
func (c *UAConfig) GetActiveSession(callID string) (*SessionInfo, bool) {
	c.activeMutex.RLock()
	defer c.activeMutex.RUnlock()
	session, exists := c.activeSessions[callID]
	if !exists {
		return nil, false
	}
	return session, true
}

// RemoveActiveSession removes an active session
func (c *UAConfig) RemoveActiveSession(callID string) {
	c.activeMutex.Lock()
	defer c.activeMutex.Unlock()
	delete(c.activeSessions, callID)
}

// ==================== Database Storage Implementation ====================

func (c *UAConfig) savePendingSessionToDatabase(callID, clientRTPAddr string) error {
	if c.Db == nil {
		// Fallback to memory if database not configured
		return c.savePendingSessionToMemory(callID, clientRTPAddr)
	}
	// Save to database
	session := &models.SipSession{
		CallID:        callID,
		Status:        models.SipSessionStatusPending,
		RemoteRTPAddr: clientRTPAddr,
		CreatedTime:   time.Now(),
	}
	return models.CreateSipSession(c.Db, session)
}

func (c *UAConfig) getPendingSessionFromDatabase(callID string) (string, bool) {
	if c.Db == nil {
		// Fallback to memory
		return c.getPendingSessionFromMemory(callID)
	}
	// Get from database
	session, err := models.GetSipSessionByCallID(c.Db, callID)
	if err != nil {
		return "", false
	}
	if session.Status != models.SipSessionStatusPending {
		return "", false
	}
	return session.RemoteRTPAddr, true
}

func (c *UAConfig) removePendingSessionFromDatabase(callID string) error {
	if c.Db == nil {
		// Fallback to memory
		return c.removePendingSessionFromMemory(callID)
	}
	// Delete from database
	return models.DeleteSipSessionByCallID(c.Db, callID)
}

func (c *UAConfig) saveCallToDatabase(sipCall *models.SipCall) error {
	if c.Db == nil {
		// Fallback to memory
		return c.saveCallToMemory(sipCall)
	}
	// Save to database
	return models.CreateSipCall(c.Db, sipCall)
}

func (c *UAConfig) getCallFromDatabase(callID string) (*models.SipCall, bool) {
	if c.Db == nil {
		// Fallback to memory
		return c.getCallFromMemory(callID)
	}
	// Get from database
	call, err := models.GetSipCallByCallID(c.Db, callID)
	if err != nil {
		return nil, false
	}
	return call, true
}

// ==================== File Storage Implementation ====================

func (c *UAConfig) savePendingSessionToFile(callID, clientRTPAddr string) error {
	if err := os.MkdirAll(c.StoragePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	sessionsDir := filepath.Join(c.StoragePath, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	filePath := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", callID))

	sessionData := map[string]interface{}{
		"callId":        callID,
		"remoteRtpAddr": clientRTPAddr,
		"status":        "pending",
		"createdTime":   time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

func (c *UAConfig) getPendingSessionFromFile(callID string) (string, bool) {
	sessionsDir := filepath.Join(c.StoragePath, "sessions")
	filePath := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", callID))

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return "", false
	}

	addr, ok := sessionData["remoteRtpAddr"].(string)
	return addr, ok
}

func (c *UAConfig) removePendingSessionFromFile(callID string) error {
	sessionsDir := filepath.Join(c.StoragePath, "sessions")
	filePath := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", callID))
	return os.Remove(filePath)
}

// ==================== Memory Storage Implementation ====================

func (c *UAConfig) savePendingSessionToMemory(callID, clientRTPAddr string) error {
	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()
	c.pendingSessions[callID] = clientRTPAddr
	return nil
}

func (c *UAConfig) getPendingSessionFromMemory(callID string) (string, bool) {
	c.sessionsMutex.RLock()
	defer c.sessionsMutex.RUnlock()
	addr, exists := c.pendingSessions[callID]
	return addr, exists
}

func (c *UAConfig) removePendingSessionFromMemory(callID string) error {
	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()
	delete(c.pendingSessions, callID)
	return nil
}

func (c *UAConfig) saveCallToMemory(sipCall *models.SipCall) error {
	c.memoryCallsMutex.Lock()
	defer c.memoryCallsMutex.Unlock()
	callCopy := *sipCall
	c.memoryCalls[sipCall.CallID] = &callCopy
	return nil
}

func (c *UAConfig) getCallFromMemory(callID string) (*models.SipCall, bool) {
	c.memoryCallsMutex.RLock()
	defer c.memoryCallsMutex.RUnlock()
	call, exists := c.memoryCalls[callID]
	if !exists {
		return nil, false
	}
	callCopy := *call
	return &callCopy, true
}

// SaveRegistrationToFile saves registration information to file
func (c *UAConfig) SaveRegistrationToFile(info *RegistrationInfo) error {
	// Ensure storage directory exists
	if err := os.MkdirAll(c.StoragePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create file path: storagePath/registrations/username.json
	registrationsDir := filepath.Join(c.StoragePath, "registrations")
	if err := os.MkdirAll(registrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create registrations directory: %w", err)
	}

	filePath := filepath.Join(registrationsDir, fmt.Sprintf("%s.json", info.Username))

	// Prepare registration data
	regData := map[string]interface{}{
		"username":     info.Username,
		"contact":      info.ContactStr,
		"contactIP":    info.ContactIP,
		"contactPort":  info.ContactPort,
		"expires":      info.Expires,
		"expiresAt":    time.Now().Add(time.Duration(info.Expires) * time.Second).Format(time.RFC3339),
		"userAgent":    info.UserAgent,
		"remoteIP":     info.RemoteIP,
		"status":       "registered",
		"lastRegister": time.Now().Format(time.RFC3339),
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(regData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registration data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write registration file: %w", err)
	}

	return nil
}

// SaveRegistrationToDatabase saves registration information to database
func (c *UAConfig) SaveRegistrationToDatabase(info *RegistrationInfo) error {
	if c.Db == nil {
		return fmt.Errorf("database not configured")
	}

	var sipUser models.SipUser
	err := c.Db.Where("username = ?", info.Username).First(&sipUser).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("SIP user not found: %s", info.Username)
		}
		return fmt.Errorf("database query failed: %w", err)
	}

	// Check if user is enabled
	if !sipUser.Enabled {
		return fmt.Errorf("SIP user is disabled: %s", info.Username)
	}

	// Update user information
	now := time.Now()
	sipUser.Contact = info.ContactStr
	sipUser.ContactIP = info.ContactIP
	sipUser.ContactPort = info.ContactPort
	sipUser.Expires = info.Expires
	sipUser.Status = models.SipUserStatusRegistered
	sipUser.LastRegister = &now
	sipUser.RegisterCount++
	sipUser.UserAgent = info.UserAgent
	sipUser.RemoteIP = info.RemoteIP
	sipUser.UpdateExpiresAt()

	// Save to database
	if err := c.Db.Save(&sipUser).Error; err != nil {
		return fmt.Errorf("failed to update SIP user in database: %w", err)
	}

	return nil
}

// SaveInviteToFile saves INVITE call information to file
func (c *UAConfig) SaveInviteToFile(sipCall *models.SipCall) error {
	// Ensure storage directory exists
	if err := os.MkdirAll(c.StoragePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create file path: storagePath/calls/callID.json
	callsDir := filepath.Join(c.StoragePath, "calls")
	if err := os.MkdirAll(callsDir, 0755); err != nil {
		return fmt.Errorf("failed to create calls directory: %w", err)
	}

	filePath := filepath.Join(callsDir, fmt.Sprintf("%s.json", sipCall.CallID))

	// Prepare call data
	callData := map[string]interface{}{
		"callId":        sipCall.CallID,
		"direction":     string(sipCall.Direction),
		"status":        string(sipCall.Status),
		"fromUsername":  sipCall.FromUsername,
		"fromUri":       sipCall.FromURI,
		"fromIp":        sipCall.FromIP,
		"toUsername":    sipCall.ToUsername,
		"toUri":         sipCall.ToURI,
		"localRtpAddr":  sipCall.LocalRTPAddr,
		"remoteRtpAddr": sipCall.RemoteRTPAddr,
		"startTime":     sipCall.StartTime.Format(time.RFC3339),
	}

	if sipCall.AnswerTime != nil {
		callData["answerTime"] = sipCall.AnswerTime.Format(time.RFC3339)
	}
	if sipCall.EndTime != nil {
		callData["endTime"] = sipCall.EndTime.Format(time.RFC3339)
	}
	if sipCall.Duration > 0 {
		callData["duration"] = sipCall.Duration
	}
	if sipCall.ErrorCode != 0 {
		callData["errorCode"] = sipCall.ErrorCode
	}
	if sipCall.ErrorMessage != "" {
		callData["errorMessage"] = sipCall.ErrorMessage
	}
	if sipCall.RecordURL != "" {
		callData["recordUrl"] = sipCall.RecordURL
	}
	if sipCall.Metadata != "" {
		callData["metadata"] = sipCall.Metadata
	}
	if sipCall.Notes != "" {
		callData["notes"] = sipCall.Notes
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(callData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal call data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write call file: %w", err)
	}

	return nil
}

// SaveInviteToDatabase saves INVITE call information to database
func (c *UAConfig) SaveInviteToDatabase(sipCall *models.SipCall) error {
	if c.Db == nil {
		return fmt.Errorf("database not configured")
	}

	if err := c.Db.Create(sipCall).Error; err != nil {
		return fmt.Errorf("failed to create SIP call in database: %w", err)
	}

	return nil
}

// UpdateCallStatusInFile updates call status in file
func (c *UAConfig) UpdateCallStatusInFile(callID string, status models.SipCallStatus, answerTime *time.Time) {
	callsDir := filepath.Join(c.StoragePath, "calls")
	filePath := filepath.Join(callsDir, fmt.Sprintf("%s.json", callID))

	data, err := os.ReadFile(filePath)
	if err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to read call file for status update")
		return
	}

	var callData map[string]interface{}
	if err := json.Unmarshal(data, &callData); err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to unmarshal call data")
		return
	}

	callData["status"] = string(status)
	if answerTime != nil {
		callData["answerTime"] = answerTime.Format(time.RFC3339)
	}

	jsonData, err := json.MarshalIndent(callData, "", "  ")
	if err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to marshal call data")
		return
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to write call file")
	} else {
		logrus.WithFields(logrus.Fields{
			"call_id": callID,
			"status":  status,
		}).Info("Call status updated in file")
	}
}
