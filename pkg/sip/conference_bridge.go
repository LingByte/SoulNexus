package sip

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/sirupsen/logrus"
)

// ConferenceBridge 会议桥接器
type ConferenceBridge struct {
	conferences map[string]*Conference // ConferenceID -> Conference
	confMutex   sync.RWMutex

	rtpConn    *net.UDPConn
	sampleRate uint32
}

// Conference 会议信息
type Conference struct {
	ID           string                  `json:"id"`
	CreatedAt    time.Time               `json:"createdAt"`
	Participants map[string]*Participant `json:"participants"` // CallID -> Participant
	partMutex    sync.RWMutex

	// 音频混合
	mixer      *AudioMixer
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// Participant 会议参与者
type Participant struct {
	CallID       string       `json:"callId"`
	RTPAddr      *net.UDPAddr `json:"rtpAddr"`
	SSRC         uint32       `json:"ssrc"`
	LastActivity time.Time    `json:"lastActivity"`
	Muted        bool         `json:"muted"`
	Volume       float64      `json:"volume"` // 0.0 - 1.0
}

// AudioMixer 音频混合器
type AudioMixer struct {
	participants map[string]*Participant
	partMutex    sync.RWMutex
	sampleRate   uint32
}

// NewConferenceBridge 创建会议桥接器
func NewConferenceBridge(rtpConn *net.UDPConn, sampleRate uint32) *ConferenceBridge {
	return &ConferenceBridge{
		conferences: make(map[string]*Conference),
		rtpConn:     rtpConn,
		sampleRate:  sampleRate,
	}
}

// CreateConference 创建新会议
func (cb *ConferenceBridge) CreateConference(conferenceID string) (*Conference, error) {
	cb.confMutex.Lock()
	defer cb.confMutex.Unlock()

	if _, exists := cb.conferences[conferenceID]; exists {
		return nil, fmt.Errorf("conference already exists: %s", conferenceID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	conference := &Conference{
		ID:           conferenceID,
		CreatedAt:    time.Now(),
		Participants: make(map[string]*Participant),
		mixer: &AudioMixer{
			participants: make(map[string]*Participant),
			sampleRate:   cb.sampleRate,
		},
		ctx:        ctx,
		cancelFunc: cancel,
	}

	cb.conferences[conferenceID] = conference

	logrus.WithField("conference_id", conferenceID).Info("Conference created")
	return conference, nil
}

// AddParticipant 添加参与者到会议
func (cb *ConferenceBridge) AddParticipant(conferenceID, callID string, rtpAddr *net.UDPAddr, ssrc uint32) error {
	cb.confMutex.RLock()
	conference, exists := cb.conferences[conferenceID]
	cb.confMutex.RUnlock()

	if !exists {
		return fmt.Errorf("conference not found: %s", conferenceID)
	}

	conference.partMutex.Lock()
	defer conference.partMutex.Unlock()

	participant := &Participant{
		CallID:       callID,
		RTPAddr:      rtpAddr,
		SSRC:         ssrc,
		LastActivity: time.Now(),
		Muted:        false,
		Volume:       1.0,
	}

	conference.Participants[callID] = participant
	conference.mixer.partMutex.Lock()
	conference.mixer.participants[callID] = participant
	conference.mixer.partMutex.Unlock()

	// 如果这是第一个参与者，启动音频混合
	if len(conference.Participants) == 1 {
		go cb.startMixing(conference)
	}

	logrus.WithFields(logrus.Fields{
		"conference_id": conferenceID,
		"call_id":       callID,
	}).Info("Participant added to conference")

	return nil
}

// RemoveParticipant 从会议中移除参与者
func (cb *ConferenceBridge) RemoveParticipant(conferenceID, callID string) error {
	cb.confMutex.RLock()
	conference, exists := cb.conferences[conferenceID]
	cb.confMutex.RUnlock()

	if !exists {
		return fmt.Errorf("conference not found: %s", conferenceID)
	}

	conference.partMutex.Lock()
	delete(conference.Participants, callID)
	participantCount := len(conference.Participants)
	conference.partMutex.Unlock()

	conference.mixer.partMutex.Lock()
	delete(conference.mixer.participants, callID)
	conference.mixer.partMutex.Unlock()

	// 如果没有参与者了，结束会议
	if participantCount == 0 {
		cb.EndConference(conferenceID)
	}

	logrus.WithFields(logrus.Fields{
		"conference_id": conferenceID,
		"call_id":       callID,
	}).Info("Participant removed from conference")

	return nil
}

// EndConference 结束会议
func (cb *ConferenceBridge) EndConference(conferenceID string) error {
	cb.confMutex.Lock()
	defer cb.confMutex.Unlock()

	conference, exists := cb.conferences[conferenceID]
	if !exists {
		return fmt.Errorf("conference not found: %s", conferenceID)
	}

	// 取消上下文，停止所有goroutine
	conference.cancelFunc()

	delete(cb.conferences, conferenceID)

	logrus.WithField("conference_id", conferenceID).Info("Conference ended")
	return nil
}

// startMixing 启动音频混合
func (cb *ConferenceBridge) startMixing(conference *Conference) {
	ticker := time.NewTicker(20 * time.Millisecond) // 50 packets per second
	defer ticker.Stop()

	packetBuffer := make(map[string][]int16) // CallID -> audio samples

	for {
		select {
		case <-conference.ctx.Done():
			return
		case <-ticker.C:
			// 收集所有参与者的音频
			conference.mixer.partMutex.RLock()
			participants := make([]*Participant, 0, len(conference.mixer.participants))
			for _, p := range conference.mixer.participants {
				participants = append(participants, p)
			}
			conference.mixer.partMutex.RUnlock()

			if len(participants) < 2 {
				continue // 需要至少2个参与者才能混合
			}

			// 混合音频
			mixedAudio := cb.mixAudio(participants, packetBuffer)

			// 发送混合后的音频给每个参与者（排除发送者）
			for _, participant := range participants {
				if !participant.Muted && participant.Volume > 0 {
					cb.sendMixedAudio(participant, mixedAudio, conference.ID)
				}
			}
		}
	}
}

// mixAudio 混合多个音频流
func (cb *ConferenceBridge) mixAudio(participants []*Participant, buffer map[string][]int16) []int16 {
	// 这里简化处理，实际应该从RTP包中提取音频数据
	// 假设每个参与者有160个样本（20ms @ 8kHz）
	samplesPerPacket := 160
	mixed := make([]int16, samplesPerPacket)

	// 收集所有参与者的音频样本
	for _, participant := range participants {
		if participant.Muted || participant.Volume <= 0 {
			continue
		}

		// 从buffer中获取音频（实际应该从RTP包中获取）
		audio, exists := buffer[participant.CallID]
		if !exists || len(audio) < samplesPerPacket {
			// 如果没有音频数据，使用静音
			audio = make([]int16, samplesPerPacket)
		}

		// 混合音频（加权平均）
		for i := 0; i < samplesPerPacket && i < len(audio); i++ {
			mixed[i] += int16(float64(audio[i]) * participant.Volume)
		}
	}

	// 防止溢出（限制在int16范围内）
	for i := range mixed {
		if mixed[i] > 32767 {
			mixed[i] = 32767
		} else if mixed[i] < -32768 {
			mixed[i] = -32768
		}
	}

	return mixed
}

// sendMixedAudio 发送混合后的音频给参与者
func (cb *ConferenceBridge) sendMixedAudio(participant *Participant, audio []int16, conferenceID string) {
	if cb.rtpConn == nil || participant.RTPAddr == nil {
		return
	}

	// 将PCM转换为μ-law
	ulawAudio := make([]byte, len(audio))
	for i, sample := range audio {
		ulawAudio[i] = linearToULaw(sample)
	}

	// 创建RTP包
	packet := &rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			PayloadType:    0, // PCMU
			SequenceNumber: 0, // 应该使用递增的序列号
			Timestamp:      0, // 应该使用递增的时间戳
			SSRC:           participant.SSRC,
		},
		Payload: ulawAudio,
	}

	// 序列化RTP包
	packetBuf, err := packet.Marshal()
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal RTP packet")
		return
	}

	// 发送RTP包
	_, err = cb.rtpConn.WriteToUDP(packetBuf, participant.RTPAddr)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"conference_id": conferenceID,
			"call_id":       participant.CallID,
		}).Error("Failed to send mixed audio")
	}
}

// GetConference 获取会议信息
func (cb *ConferenceBridge) GetConference(conferenceID string) (*Conference, bool) {
	cb.confMutex.RLock()
	defer cb.confMutex.RUnlock()

	conference, exists := cb.conferences[conferenceID]
	return conference, exists
}

// GetAllConferences 获取所有会议
func (cb *ConferenceBridge) GetAllConferences() map[string]*Conference {
	cb.confMutex.RLock()
	defer cb.confMutex.RUnlock()

	result := make(map[string]*Conference)
	for id, conf := range cb.conferences {
		result[id] = conf
	}
	return result
}

// MuteParticipant 静音参与者
func (cb *ConferenceBridge) MuteParticipant(conferenceID, callID string, muted bool) error {
	cb.confMutex.RLock()
	conference, exists := cb.conferences[conferenceID]
	cb.confMutex.RUnlock()

	if !exists {
		return fmt.Errorf("conference not found: %s", conferenceID)
	}

	conference.partMutex.Lock()
	defer conference.partMutex.Unlock()

	participant, exists := conference.Participants[callID]
	if !exists {
		return fmt.Errorf("participant not found: %s", callID)
	}

	participant.Muted = muted
	return nil
}

// SetParticipantVolume 设置参与者音量
func (cb *ConferenceBridge) SetParticipantVolume(conferenceID, callID string, volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0.0 and 1.0")
	}

	cb.confMutex.RLock()
	conference, exists := cb.conferences[conferenceID]
	cb.confMutex.RUnlock()

	if !exists {
		return fmt.Errorf("conference not found: %s", conferenceID)
	}

	conference.partMutex.Lock()
	defer conference.partMutex.Unlock()

	participant, exists := conference.Participants[callID]
	if !exists {
		return fmt.Errorf("participant not found: %s", callID)
	}

	participant.Volume = volume
	return nil
}

// linearToULaw 将线性PCM转换为μ-law
func linearToULaw(sample int16) byte {
	var sign, exponent, mantissa, ulawbyte byte

	sample = sample >> 2
	if sample < 0 {
		sample = -sample
		sign = 0x80
	} else {
		sign = 0x00
	}

	if sample > 32635 {
		sample = 32635
	}

	sample += 0x84

	exponent = expLut[(sample>>7)&0xFF]
	mantissa = byte((sample >> (exponent + 3)) & 0x0F)
	ulawbyte = ^(sign | (exponent << 4) | mantissa)

	return ulawbyte
}

// expLut μ-law编码的指数查找表
var expLut = [256]byte{
	0, 1, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4,
	5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
}
