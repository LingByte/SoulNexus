package sip

import (
	"context"
	"time"
)

// EventType represents the type of SIP event
type EventType string

const (
	EventTypePlayAudio EventType = "play_audio" // Play audio event
	EventTypeRecord    EventType = "record"     // Record audio event
	EventTypeDTMF      EventType = "dtmf"       // DTMF key event
)

// SipEvent is the base interface for all SIP events
type SipEvent interface {
	Type() EventType
	CallID() string
	Context() context.Context
	Execute(server *SipServer) error
}

// PlayAudioEvent represents an audio playback event
type PlayAudioEvent struct {
	callID           string
	ctx              context.Context
	ClientAddr       string
	Filename         string
	SampleRate       uint32
	SamplesPerPacket int
}

// NewPlayAudioEvent creates a new PlayAudioEvent
func NewPlayAudioEvent(callID string, ctx context.Context, clientAddr, filename string, sampleRate uint32, samplesPerPacket int) *PlayAudioEvent {
	return &PlayAudioEvent{
		callID:           callID,
		ctx:              ctx,
		ClientAddr:       clientAddr,
		Filename:         filename,
		SampleRate:       sampleRate,
		SamplesPerPacket: samplesPerPacket,
	}
}

func (e *PlayAudioEvent) Type() EventType {
	return EventTypePlayAudio
}

func (e *PlayAudioEvent) CallID() string {
	return e.callID
}

func (e *PlayAudioEvent) Context() context.Context {
	return e.ctx
}

func (e *PlayAudioEvent) Execute(server *SipServer) error {
	if e.Filename != "" {
		// Play from file
		server.sendAudioFromFileWithContext(e.ClientAddr, e.Filename, e.SamplesPerPacket, e.ctx)
	} else {
		// Play default audio
		server.sendAudioWithContext(e.ClientAddr, e.SampleRate, e.SamplesPerPacket, e.ctx)
	}
	return nil
}

// RecordAudioEvent represents an audio recording event
type RecordAudioEvent struct {
	callID      string
	ctx         context.Context
	ClientAddr  string
	Filename    string
	Duration    time.Duration
	SampleRate  int
	StopChannel chan bool
}

// NewRecordAudioEvent creates a new RecordAudioEvent
func NewRecordAudioEvent(callID string, ctx context.Context, clientAddr, filename string, duration time.Duration, sampleRate int, stopChannel chan bool) *RecordAudioEvent {
	return &RecordAudioEvent{
		callID:      callID,
		ctx:         ctx,
		ClientAddr:  clientAddr,
		Filename:    filename,
		Duration:    duration,
		SampleRate:  sampleRate,
		StopChannel: stopChannel,
	}
}

func (e *RecordAudioEvent) Type() EventType {
	return EventTypeRecord
}

func (e *RecordAudioEvent) CallID() string {
	return e.callID
}

func (e *RecordAudioEvent) Context() context.Context {
	return e.ctx
}

func (e *RecordAudioEvent) Execute(server *SipServer) error {
	server.recordAudioWithContext(e.ClientAddr, e.Filename, e.Duration, e.SampleRate, e.ctx, e.StopChannel)
	return nil
}

// DTMFEvent represents a DTMF key event
type DTMFEvent struct {
	callID     string
	ctx        context.Context
	ClientAddr string
	Key        string
	Action     DTMFAction
}

// DTMFAction represents the action to take when a DTMF key is pressed
type DTMFAction struct {
	Key      string
	Filename string
}

// NewDTMFEvent creates a new DTMFEvent
func NewDTMFEvent(callID string, ctx context.Context, clientAddr, key string) *DTMFEvent {
	return &DTMFEvent{
		callID:     callID,
		ctx:        ctx,
		ClientAddr: clientAddr,
		Key:        key,
	}
}

// WithAction sets the action for the DTMF event
func (e *DTMFEvent) WithAction(filename string) *DTMFEvent {
	e.Action = DTMFAction{
		Key:      e.Key,
		Filename: filename,
	}
	return e
}

func (e *DTMFEvent) Type() EventType {
	return EventTypeDTMF
}

func (e *DTMFEvent) CallID() string {
	return e.callID
}

func (e *DTMFEvent) Context() context.Context {
	return e.ctx
}

func (e *DTMFEvent) Execute(server *SipServer) error {
	// Execute DTMF action (play corresponding audio file)
	if e.Action.Filename != "" {
		server.sendAudioFromFileWithContext(e.ClientAddr, e.Action.Filename, 160, e.ctx)
	}
	return nil
}

// EventHandler processes SIP events
type EventHandler interface {
	Handle(event SipEvent) error
}

// EventProcessor processes events sequentially
type EventProcessor struct {
	server *SipServer
}

// NewEventProcessor creates a new EventProcessor
func NewEventProcessor(server *SipServer) *EventProcessor {
	return &EventProcessor{
		server: server,
	}
}

// Process executes an event
func (p *EventProcessor) Process(event SipEvent) error {
	// Check if context is cancelled before processing
	select {
	case <-event.Context().Done():
		return event.Context().Err()
	default:
	}

	return event.Execute(p.server)
}

// ProcessSequence processes a sequence of events
func (p *EventProcessor) ProcessSequence(events []SipEvent) error {
	for _, event := range events {
		// Check if context is cancelled before processing each event
		select {
		case <-event.Context().Done():
			return event.Context().Err()
		default:
		}

		if err := event.Execute(p.server); err != nil {
			return err
		}
	}
	return nil
}
