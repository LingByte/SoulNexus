package sip1

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/sip1/ua"
	"github.com/emiago/sipgo/sip"
	"github.com/pion/rtp"
	"github.com/sirupsen/logrus"
)

func (as *SipServer) RegisterFunc() {
	as.server.OnRegister(as.handleRegister)
	as.server.OnInvite(as.handleInvite)
	as.server.OnOptions(as.handleOptions)
	as.server.OnAck(as.handleAck)
}

// handleRegister handles SIP REGISTER requests based on configured storage type
func (as *SipServer) handleRegister(req *sip.Request, tx sip.ServerTransaction) {
	logrus.WithField("start_line", req.StartLine()).Info("Received REGISTER request")

	// Extract registration information from request
	info := as.config.ExtractRegistrationInfo(req)

	// Validate username
	if info.Username == "" {
		logrus.Warn("REGISTER request missing username in From header")
		res := sip.NewResponseFromRequest(req, sip.StatusUnauthorized, "Unauthorized", nil)
		if err := tx.Respond(res); err != nil {
			logrus.WithError(err).Error("Failed to send 401 response")
		}
		return
	}

	// Process based on storage type configuration
	var saveErr error
	switch as.config.StorageType {
	case ua.StorageTypeDatabase:
		logrus.WithField("username", info.Username).Info("Processing registration with database storage")

		if as.config.Db == nil {
			logrus.Warn("Database not configured for database storage type, falling back to memory storage")
			if info.ContactStr != "" {
				contact := fmt.Sprintf("%s:%d", info.ContactIP, info.ContactPort)
				as.config.SetRegisteredUser(info.Username, contact)
			}
			break
		}

		// Save to database
		saveErr = as.config.SaveRegistrationToDatabase(info)
		if saveErr != nil {
			logrus.WithError(saveErr).Error("Failed to save registration to database")
			// Determine error type and return appropriate response
			errMsg := saveErr.Error()
			if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "disabled") {
				status := sip.StatusUnauthorized
				statusText := "Unauthorized"
				if strings.Contains(errMsg, "disabled") {
					status = sip.StatusForbidden
					statusText = "Forbidden"
				}
				res := sip.NewResponseFromRequest(req, status, statusText, nil)
				if err := tx.Respond(res); err != nil {
					logrus.WithError(err).Error("Failed to send response")
				}
				return
			}
			// Database error
			res := sip.NewResponseFromRequest(req, sip.StatusInternalServerError, "Internal Server Error", nil)
			if err := tx.Respond(res); err != nil {
				logrus.WithError(err).Error("Failed to send 500 response")
			}
			return
		}

		logrus.WithFields(logrus.Fields{
			"username": info.Username,
			"contact":  info.ContactStr,
			"expires":  info.Expires,
		}).Info("SIP user registered successfully in database")

	case ua.StorageTypeFile:
		logrus.WithField("username", info.Username).Info("Processing registration with file storage")

		// Save to file
		saveErr = as.config.SaveRegistrationToFile(info)
		if saveErr != nil {
			logrus.WithError(saveErr).Error("Failed to save registration to file")
			res := sip.NewResponseFromRequest(req, sip.StatusInternalServerError, "Internal Server Error", nil)
			if err := tx.Respond(res); err != nil {
				logrus.WithError(err).Error("Failed to send 500 response")
			}
			return
		}

		logrus.WithFields(logrus.Fields{
			"username": info.Username,
			"contact":  info.ContactStr,
			"expires":  info.Expires,
		}).Info("SIP user registered successfully in file storage")

	case ua.StorageTypeMemory:
		logrus.WithField("username", info.Username).Info("Processing registration with memory storage")

		if info.ContactStr != "" {
			contact := fmt.Sprintf("%s:%d", info.ContactIP, info.ContactPort)
			as.config.SetRegisteredUser(info.Username, contact)
		}

		logrus.WithFields(logrus.Fields{
			"username": info.Username,
			"contact":  info.ContactStr,
			"expires":  info.Expires,
		}).Info("SIP user registered successfully in memory storage")

	default:
		logrus.Warnf("Unknown storage type: %s, defaulting to memory storage", as.config.StorageType)
		logrus.WithFields(logrus.Fields{
			"username": info.Username,
			"contact":  info.ContactStr,
			"expires":  info.Expires,
		}).Info("SIP user registered successfully in memory storage (default)")
	}

	// Accept registration, return 200 OK
	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)

	// Add Contact header (optional)
	if contact := req.Contact(); contact != nil {
		res.AppendHeader(contact)
	}

	// Add Expires header using the extracted expires value
	expires := sip.ExpiresHeader(info.Expires)
	res.AppendHeader(&expires)

	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send REGISTER response")
		return
	}

	logrus.Info("REGISTER 200 OK response sent")
}

func (as *SipServer) handleInvite(req *sip.Request, tx sip.ServerTransaction) {
	logrus.WithField("start_line", req.StartLine()).Info("Received INVITE request")

	// Parse SDP to get client RTP address
	sdpBody := string(req.Body())
	clientRTPAddr, err := ParseSDPForRTPAddress(sdpBody)
	if err != nil {
		logrus.WithError(err).Error("Failed to parse SDP")
		// Send 500 error response
		res := sip.NewResponseFromRequest(req, sip.StatusInternalServerError, "Internal Server Error", nil)
		tx.Respond(res)
		return
	}

	logrus.WithField("client_rtp_addr", clientRTPAddr).Info("Client RTP address")

	// Generate SDP response (use request source address to determine server IP)
	serverIP := getServerIPFromRequest(req)
	sdp := generateSDP(serverIP, as.config.LocalRTPPort)
	sdpBytes := []byte(sdp)

	// Log SDP content for debugging
	logrus.WithField("sdp", sdp).Debug("Generated SDP")

	// Create 200 OK response
	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", sdpBytes)
	cl := sip.ContentLengthHeader(len(sdpBytes))
	res.AppendHeader(&cl)

	// Add Content-Type header
	contentType := sip.ContentTypeHeader("application/sdp")
	res.AppendHeader(&contentType)

	// Add Contact header (some clients need this to send ACK correctly)
	// Create a Contact header using server IP and port
	contactURI := sip.Uri{
		Host: serverIP,
		Port: as.config.Port,
	}
	contact := &sip.ContactHeader{
		Address: contactURI,
	}
	res.AppendHeader(contact)
	logrus.WithField("contact", contact.String()).Debug("Contact header")

	// Send 200 OK response
	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send response")
		return
	}

	logrus.Info("200 OK response sent with SDP and Contact header")
	logrus.Info("200 OK response sent, waiting for ACK...")

	// Save session information, wait for ACK before sending audio
	callID := req.CallID().Value()
	if err := as.config.SavePendingSession(callID, clientRTPAddr); err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to save pending session")
	} else {
		logrus.WithFields(logrus.Fields{
			"call_id":     callID,
			"rtp_address": clientRTPAddr,
		}).Info("Session information saved")
	}

	// Extract call information from request
	now := time.Now()
	from := req.From()
	to := req.To()

	var fromUsername, fromURI, fromIP string
	var toUsername, toURI string

	if from != nil {
		fromUsername = from.Address.User
		fromURI = from.Address.String()
		// 从请求中获取源IP
		if via := req.Via(); via != nil {
			fromIP = via.Host
		}
	}

	if to != nil {
		toUsername = to.Address.User
		toURI = to.Address.String()
	}

	localRTPAddr := fmt.Sprintf("%s:%d", serverIP, as.config.LocalRTPPort)

	sipCall := &models.SipCall{
		CallID:        callID,
		Direction:     models.SipCallDirectionInbound,
		Status:        models.SipCallStatusRinging,
		FromUsername:  fromUsername,
		FromURI:       fromURI,
		FromIP:        fromIP,
		ToUsername:    toUsername,
		ToURI:         toURI,
		LocalRTPAddr:  localRTPAddr,
		RemoteRTPAddr: clientRTPAddr,
		StartTime:     now,
	}

	// Process based on storage type configuration
	var saveErr error
	switch as.config.StorageType {
	case ua.StorageTypeDatabase:
		logrus.WithField("call_id", callID).Info("Processing INVITE with database storage")

		if as.config.Db == nil {
			logrus.Warn("Database not configured for database storage type, falling back to memory storage")
			saveErr = as.config.SaveCall(sipCall)
			break
		}

		// Save to database
		saveErr = as.config.SaveInviteToDatabase(sipCall)
		if saveErr != nil {
			logrus.WithError(saveErr).WithField("call_id", callID).Error("Failed to save INVITE to database")
		} else {
			logrus.WithField("call_id", callID).Info("Inbound call record created in database")
		}

	case ua.StorageTypeFile:
		logrus.WithField("call_id", callID).Info("Processing INVITE with file storage")

		// Save to file
		saveErr = as.config.SaveInviteToFile(sipCall)
		if saveErr != nil {
			logrus.WithError(saveErr).WithField("call_id", callID).Error("Failed to save INVITE to file")
		} else {
			logrus.WithField("call_id", callID).Info("Inbound call record created in file storage")
		}

	case ua.StorageTypeMemory:
		logrus.WithField("call_id", callID).Info("Processing INVITE with memory storage")

		// Save to memory only
		saveErr = as.config.SaveCall(sipCall)
		if saveErr != nil {
			logrus.WithError(saveErr).WithField("call_id", callID).Error("Failed to save INVITE to memory")
		} else {
			logrus.WithField("call_id", callID).Info("Inbound call record created in memory storage")
		}

	default:
		logrus.Warnf("Unknown storage type: %s, defaulting to memory storage", as.config.StorageType)
		saveErr = as.config.SaveCall(sipCall)
		if saveErr != nil {
			logrus.WithError(saveErr).WithField("call_id", callID).Error("Failed to save INVITE to memory")
		} else {
			logrus.WithField("call_id", callID).Info("Inbound call record created in memory storage (default)")
		}
	}
}

func (as *SipServer) handleOptions(req *sip.Request, tx sip.ServerTransaction) {
	logrus.WithField("start_line", req.StartLine()).Info("Received OPTIONS request")

	// Return 200 OK, indicating support for these methods
	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)

	// Add Allow header, list supported methods
	allow := sip.NewHeader("Allow", "INVITE, ACK, CANCEL, BYE, OPTIONS, REGISTER")
	res.AppendHeader(allow)

	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send OPTIONS response")
		return
	}

	logrus.Info("OPTIONS 200 OK response sent")
}

func (as *SipServer) handleAck(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	logrus.WithFields(logrus.Fields{
		"start_line": req.StartLine(),
		"call_id":    callID,
	}).Info("Received ACK request")

	// ACK request doesn't need a response, but receiving ACK means session is established, can start sending audio
	// Find corresponding session information using config methods
	clientRTPAddr, exists := as.config.GetPendingSession(callID)
	if !exists {
		logrus.WithField("call_id", callID).Warn("Received ACK but could not find corresponding session")
		return
	}

	// Remove pending session
	if err := as.config.RemovePendingSession(callID); err != nil {
		logrus.WithError(err).WithField("call_id", callID).Warn("Failed to remove pending session")
	}

	// Save active session information
	clientAddr, err := net.ResolveUDPAddr("udp", clientRTPAddr)
	if err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to resolve client address")
		return
	}
	logrus.WithFields(logrus.Fields{
		"call_id":         callID,
		"client_rtp_addr": clientRTPAddr,
	}).Info("Session established, starting to send audio")

	// 创建录音文件路径
	recordDir := "uploads/audio"
	if err := os.MkdirAll(recordDir, 0755); err != nil {
		logrus.WithError(err).Error("Failed to create audio directory")
	}
	recordingFile := fmt.Sprintf("%s/recorded_%s.wav", recordDir, callID)

	// Save active session to config
	as.config.SaveActiveSession(callID, &ua.SessionInfo{
		ClientRTPAddr: clientAddr,
		StopRecording: make(chan bool, 1),
		DTMFChannel:   make(chan string, 10), // DTMF channel
		RecordingFile: recordingFile,
	})

	// 更新数据库状态为已接通（呼入通话）
	if as.config.Db != nil {
		now := time.Now()
		as.updateCallStatusInDB(callID, models.SipCallStatusAnswered, &now)
	} else {
		// 如果使用文件或内存存储，也需要更新状态
		as.updateCallStatus(callID, models.SipCallStatusAnswered, nil)
	}

	// 启动录音（持续录音直到通话结束）
	// 使用 StopRecording channel 替代 context
	go as.recordAudioContinuous(clientRTPAddr, callID, recordingFile)

	// Send audio in goroutine
	go as.sendAudioWithCallback(clientRTPAddr, callID)
}

// saveWAV 将 PCM 数据保存为 WAV 文件
func saveWAV(filename string, pcmData []int16, sampleRate int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// WAV 文件头
	dataSize := uint32(len(pcmData) * 2) // 每个样本 2 字节
	fileSize := 36 + dataSize

	// RIFF 头
	file.WriteString("RIFF")
	binary.Write(file, binary.LittleEndian, uint32(fileSize))
	file.WriteString("WAVE")

	// fmt chunk
	file.WriteString("fmt ")
	binary.Write(file, binary.LittleEndian, uint32(16)) // fmt chunk size
	binary.Write(file, binary.LittleEndian, uint16(1))  // audio format (PCM)
	binary.Write(file, binary.LittleEndian, uint16(1))  // num channels
	binary.Write(file, binary.LittleEndian, uint32(sampleRate))
	binary.Write(file, binary.LittleEndian, uint32(sampleRate*2)) // byte rate
	binary.Write(file, binary.LittleEndian, uint16(2))            // block align
	binary.Write(file, binary.LittleEndian, uint16(16))           // bits per sample

	// data chunk
	file.WriteString("data")
	binary.Write(file, binary.LittleEndian, dataSize)

	// 写入 PCM 数据
	for _, sample := range pcmData {
		binary.Write(file, binary.LittleEndian, sample)
	}

	return nil
}

// recordAudioContinuous 持续录音（不限制时长，直到收到停止信号）
func (as *SipServer) recordAudioContinuous(clientAddr string, callID string, filename string) {
	addr, err := net.ResolveUDPAddr("udp", clientAddr)
	if err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to resolve client address")
		return
	}

	logrus.WithFields(logrus.Fields{
		"call_id":  callID,
		"filename": filename,
		"address":  addr.String(),
	}).Info("Starting continuous recording")

	// 获取 session 的 StopRecording channel
	session, exists := as.config.GetActiveSession(callID)
	if !exists {
		logrus.WithField("call_id", callID).Warn("Session not found for recording")
		return
	}

	// 创建缓冲区存储 PCM 数据
	var pcmData []int16
	buffer := make([]byte, 1500)
	packetCount := 0
	sampleRate := 8000

	// 设置读取超时（用于定期检查停止信号）
	as.rtpConn.SetReadDeadline(time.Now().Add(1 * time.Second))

	for {
		// 检查是否停止
		select {
		case <-session.StopRecording:
			logrus.WithField("call_id", callID).Info("Recording stopped")
			as.rtpConn.SetReadDeadline(time.Time{}) // Clear timeout
			// 保存录音
			if len(pcmData) > 0 {
				if err := saveWAV(filename, pcmData, sampleRate); err != nil {
					logrus.WithError(err).WithField("call_id", callID).Error("Failed to save WAV file")
				} else {
					logrus.WithFields(logrus.Fields{
						"call_id":      callID,
						"filename":     filename,
						"samples":      len(pcmData),
						"packet_count": packetCount,
					}).Info("Recording saved")
				}
			}
			return
		default:
		}

		// 动态更新超时（用于定期检查停止信号）
		as.rtpConn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, receivedAddr, err := as.rtpConn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时是正常的，继续循环检查停止信号
				continue
			}
			logrus.WithError(err).WithField("call_id", callID).Error("Failed to read RTP data")
			continue
		}

		// 检查是否来自目标客户端
		if !receivedAddr.IP.Equal(addr.IP) {
			continue
		}

		// 解析 RTP 包
		packet := &rtp.Packet{}
		if err := packet.Unmarshal(buffer[:n]); err != nil {
			logrus.WithError(err).WithField("call_id", callID).Error("Failed to parse RTP packet")
			continue
		}

		// 只处理 PCMU (payload type 0)
		if packet.PayloadType != 0 {
			continue
		}

		packetCount++

		// 解码 μ-law 为 PCM
		for _, mulawByte := range packet.Payload {
			pcm := mulawToLinear(mulawByte)
			pcmData = append(pcmData, pcm)
		}
	}
}

func (as *SipServer) sendAudioWithCallback(clientAddr string, callID string) {
	// Get session for cancellation check
	session, exists := as.config.GetActiveSession(callID)
	if !exists {
		logrus.WithField("call_id", callID).Warn("Session not found, aborting audio callback")
		return
	}

	// 使用 StopRecording channel 来检查是否停止
	// 这里可以添加实际的音频发送逻辑
	// 当需要停止时，可以通过向 StopRecording channel 发送信号来停止
	logrus.WithField("call_id", callID).Info("Audio callback started (placeholder - implement actual audio sending logic)")
	// TODO: Implement actual audio sending logic here
	// 可以使用 session.StopRecording channel 来检查是否需要停止
	_ = session // 避免未使用变量警告
}

// updateCallStatusInDB updates call status in database
func (as *SipServer) updateCallStatusInDB(callID string, status models.SipCallStatus, answerTime *time.Time) {
	if as.config.Db == nil {
		return
	}

	var sipCall models.SipCall
	if err := as.config.Db.Where("call_id = ?", callID).First(&sipCall).Error; err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to find call record for status update")
		return
	}

	sipCall.Status = status
	if answerTime != nil {
		sipCall.AnswerTime = answerTime
	}

	if err := as.config.Db.Save(&sipCall).Error; err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to update call status in database")
	} else {
		logrus.WithFields(logrus.Fields{
			"call_id": callID,
			"status":  status,
		}).Info("Call status updated in database")
	}
}

// updateCallStatus updates call status based on storage type
func (as *SipServer) updateCallStatus(callID string, status models.SipCallStatus, answerTime *time.Time) {
	switch as.config.StorageType {
	case ua.StorageTypeDatabase:
		as.updateCallStatusInDB(callID, status, answerTime)

	case ua.StorageTypeFile:
		// Update call file
		as.config.UpdateCallStatusInFile(callID, status, answerTime)

	case ua.StorageTypeMemory:
		as.config.UpdateCallStatusInMemory(callID, status, answerTime)
	default:
		// Default to memory
		as.config.UpdateCallStatusInMemory(callID, status, answerTime)
	}
}

// handleInfo handles SIP INFO request (for receiving DTMF)
func (as *SipServer) handleInfo(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	logrus.WithFields(logrus.Fields{
		"start_line": req.StartLine(),
		"call_id":    callID,
	}).Info("Received INFO request")

	// Parse DTMF information
	body := string(req.Body())
	logrus.WithField("body", body).Debug("INFO request body")

	// Find DTMF signal (usually in Signal or Key parameter)
	dtmfDigit := ""
	if strings.Contains(body, "Signal=") {
		// Parse Signal=1 format
		parts := strings.Split(body, "Signal=")
		if len(parts) > 1 {
			dtmfDigit = strings.TrimSpace(strings.Split(parts[1], "\r\n")[0])
			dtmfDigit = strings.Trim(dtmfDigit, "\"")
		}
	} else if strings.Contains(body, "key=") {
		// Parse key=1 format
		parts := strings.Split(body, "key=")
		if len(parts) > 1 {
			dtmfDigit = strings.TrimSpace(strings.Split(parts[1], "\r\n")[0])
			dtmfDigit = strings.Trim(dtmfDigit, "\"")
		}
	}

	// If not found, try to parse from Content-Type and body
	if dtmfDigit == "" && body != "" {
		// Try to extract digit directly
		for _, char := range body {
			if char >= '0' && char <= '9' {
				dtmfDigit = string(char)
				break
			}
		}
	}

	if dtmfDigit != "" {
		logrus.WithFields(logrus.Fields{
			"dtmf":    dtmfDigit,
			"call_id": callID,
		}).Info("Detected DTMF key")

		// Send DTMF to session channel
		session, exists := as.config.GetActiveSession(callID)
		if exists {
			select {
			case session.DTMFChannel <- dtmfDigit:
				logrus.WithField("dtmf", dtmfDigit).Debug("DTMF key sent to session channel")
			default:
				logrus.WithField("dtmf", dtmfDigit).Warn("DTMF channel full, dropping key")
			}
		}
	}

	// Return 200 OK
	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send INFO response")
		return
	}

	logrus.Info("INFO 200 OK response sent")
}

func (as *SipServer) handleCancel(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	logrus.WithFields(logrus.Fields{
		"start_line": req.StartLine(),
		"call_id":    callID,
	}).Info("Received CANCEL request")

	// Clean up pending session (CANCEL is sent before ACK)
	clientRTPAddr, exists := as.config.GetPendingSession(callID)
	if exists {
		logrus.WithFields(logrus.Fields{
			"call_id":     callID,
			"rtp_address": clientRTPAddr,
		}).Warn("Found pending session when receiving CANCEL, call was cancelled before ACK")
		if err := as.config.RemovePendingSession(callID); err != nil {
			logrus.WithError(err).WithField("call_id", callID).Warn("Failed to remove pending session")
		}
	}

	// Also check active sessions (in case ACK was already received)
	session, exists := as.config.GetActiveSession(callID)
	if exists {
		logrus.WithField("call_id", callID).Info("Terminating active session due to CANCEL")

		// Signal stop recording
		select {
		case session.StopRecording <- true:
		default:
		}

		// Close DTMF channel
		close(session.DTMFChannel)

		// Remove from active sessions
		as.config.RemoveActiveSession(callID)
		logrus.WithField("call_id", callID).Info("Active session terminated due to CANCEL")
	}

	// Return 200 OK for CANCEL
	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send CANCEL response")
		return
	}

	logrus.Info("CANCEL 200 OK response sent")
}

func (as *SipServer) handleBye(req *sip.Request, tx sip.ServerTransaction) {
	callID := req.CallID().Value()
	logrus.WithFields(logrus.Fields{
		"start_line": req.StartLine(),
		"call_id":    callID,
	}).Info("Received BYE request")

	now := time.Now()

	// 更新通话状态为已结束
	as.updateCallStatus(callID, models.SipCallStatusEnded, &now)

	// Clean up pending session
	clientRTPAddr, exists := as.config.GetPendingSession(callID)
	if exists {
		logrus.WithFields(logrus.Fields{
			"call_id":     callID,
			"rtp_address": clientRTPAddr,
		}).Warn("Found pending session when receiving BYE, client may have hung up early")
		if err := as.config.RemovePendingSession(callID); err != nil {
			logrus.WithError(err).WithField("call_id", callID).Warn("Failed to remove pending session")
		}
	}

	// Clean up active session and stop all operations
	var recordingFile string
	session, exists := as.config.GetActiveSession(callID)
	if exists {
		logrus.WithField("call_id", callID).Info("Terminating active session")

		// 保存录音文件路径
		recordingFile = session.RecordingFile

		// Signal stop recording
		select {
		case session.StopRecording <- true:
		default:
		}

		// Close DTMF channel
		close(session.DTMFChannel)

		// Remove from active sessions
		as.config.RemoveActiveSession(callID)
		logrus.WithField("call_id", callID).Info("Active session terminated and cleaned up")
	}

	// 等待一小段时间确保录音已保存
	if recordingFile != "" {
		time.Sleep(500 * time.Millisecond)
		// 生成录音URL并保存到数据库
		as.saveRecordingURL(callID, recordingFile)
	}

	// Return 200 OK
	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send BYE response")
		return
	}

	logrus.Info("BYE 200 OK response sent")
}

func (as *SipServer) handlePublish(req *sip.Request, tx sip.ServerTransaction) {
	logrus.WithField("start_line", req.StartLine()).Info("Received PUBLISH request")

	// Return 200 OK (accept PUBLISH request)
	res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
	if err := tx.Respond(res); err != nil {
		logrus.WithError(err).Error("Failed to send PUBLISH response")
		return
	}

	logrus.Info("PUBLISH 200 OK response sent")
}

func (as *SipServer) handleNoRoute(req *sip.Request, tx sip.ServerTransaction) {
	logrus.WithFields(logrus.Fields{
		"start_line": req.StartLine(),
		"method":     req.Method,
		"call_id":    req.CallID().Value(),
	}).Info("Received unmatched request")

	// If it's an ACK request but wasn't caught by OnAck, handle it manually
	if req.IsAck() {
		logrus.Info("Detected ACK request (via NoRoute), attempting to handle...")
		as.handleAck(req, tx)
		return
	}

	// For other unmatched requests, return 501 Not Implemented
	res := sip.NewResponseFromRequest(req, sip.StatusNotImplemented, "Not Implemented", nil)
	tx.Respond(res)
}

// saveRecordingURL 保存录音URL到数据库
func (as *SipServer) saveRecordingURL(callID string, recordingFile string) {
	if as.config.Db == nil {
		logrus.WithField("call_id", callID).Warn("Database not configured, skipping recording URL save")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(recordingFile); os.IsNotExist(err) {
		logrus.WithField("call_id", callID).WithField("file", recordingFile).Warn("Recording file does not exist")
		return
	}

	// 生成录音URL（相对路径，前端可以通过API访问）
	recordURL := fmt.Sprintf("/api/files/audio/%s", strings.TrimPrefix(recordingFile, "uploads/audio/"))

	// 更新数据库记录
	var sipCall models.SipCall
	if err := as.config.Db.Where("call_id = ?", callID).First(&sipCall).Error; err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to find call record for recording URL")
		return
	}

	sipCall.RecordURL = recordURL
	if err := as.config.Db.Save(&sipCall).Error; err != nil {
		logrus.WithError(err).WithField("call_id", callID).Error("Failed to save recording URL")
	} else {
		logrus.WithFields(logrus.Fields{
			"call_id":    callID,
			"record_url": recordURL,
		}).Info("Recording URL saved to database")
	}
}
