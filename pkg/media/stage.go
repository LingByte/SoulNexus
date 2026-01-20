package media

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PipelineStage represents a single stage in the processing pipeline
// Uses asynchronous event-driven architecture instead of synchronous chain
type PipelineStage struct {
	index         int
	session       *MediaSession
	middleware    MediaHandlerFunc
	eventQueue    chan *MediaData
	nextStage     *PipelineStage
	preFilter     PacketFilter
	postFilter    PacketFilter
	workerRunning bool
	workerWg      sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

func (sp *PipelineStage) String() string {
	isTerminal := sp.middleware == nil
	hasNext := sp.nextStage != nil
	return fmt.Sprintf("PipelineStage{index: %d, isTerminal:%v hasNext: %v, async: %v}", sp.index, isTerminal, hasNext, sp.workerRunning)
}

func (sp *PipelineStage) GetContext() context.Context {
	if sp.ctx != nil {
		return sp.ctx
	}
	return sp.session.GetContext()
}

func (sp *PipelineStage) GetSession() *MediaSession {
	return sp.session
}

// InjectPacket sets pre-processing filter function
func (sp *PipelineStage) InjectPacket(f PacketFilter) {
	sp.preFilter = f
}

func (sp *PipelineStage) CauseError(sender any, err error) {
	sp.session.CauseError(sender, err)
}

func (sp *PipelineStage) EmitState(sender any, state string, params ...any) {
	sp.session.EmitState(sender, state, params...)
}

// EmitPacket enqueues packet for asynchronous processing
func (sp *PipelineStage) EmitPacket(sender any, packet MediaPacket) {
	if sp.workerRunning {
		sp.enqueueEvent(&MediaData{
			Type:      MediaDataTypePacket,
			Packet:    packet,
			CreatedAt: time.Now(),
			Sender:    sender,
		})
	} else {
		// Fallback to synchronous processing if worker not started
		sp.processPacketAsync(sender, packet)
	}
}

func (sp *PipelineStage) AddMetric(key string, duration time.Duration) {
	sp.session.AddMetric(key, duration)
}

func (sp *PipelineStage) SendToOutput(sender any, packet MediaPacket) {
	sp.session.putPacket(DirectionOutput, packet)
}

// startWorker starts asynchronous event processing for this stage
func (sp *PipelineStage) startWorker(queueSize int) {
	if sp.workerRunning {
		return
	}
	sp.workerRunning = true
	sp.eventQueue = make(chan *MediaData, queueSize)
	sp.ctx, sp.cancel = context.WithCancel(sp.session.GetContext())

	sp.workerWg.Add(1)
	go sp.eventLoop()
}

// stopWorker stops the asynchronous worker
func (sp *PipelineStage) stopWorker() {
	if !sp.workerRunning {
		return
	}
	if sp.cancel != nil {
		sp.cancel()
	}
	close(sp.eventQueue)
	sp.workerWg.Wait()
	sp.workerRunning = false
}

// eventLoop processes events asynchronously
func (sp *PipelineStage) eventLoop() {
	defer sp.workerWg.Done()
	for {
		select {
		case <-sp.ctx.Done():
			return
		case data, ok := <-sp.eventQueue:
			if !ok {
				return
			}
			sp.processEvent(data)
		}
	}
}

// processEvent handles a single event asynchronously
func (sp *PipelineStage) processEvent(data *MediaData) {
	if data.Type == MediaDataTypePacket {
		sp.processPacketAsync(data.Sender, data.Packet)
	} else if data.Type == MediaDataTypeState {
		if sp.middleware != nil {
			sp.middleware(sp, *data)
		}
	}
}

// processPacketAsync processes packet asynchronously
func (sp *PipelineStage) processPacketAsync(sender any, packet MediaPacket) {
	// Apply pre-filter if exists
	if sp.preFilter != nil {
		shouldSkip, err := sp.preFilter(packet)
		if err != nil {
			sp.CauseError(sp, err)
			return
		}
		if shouldSkip {
			return
		}
	}

	// Terminal stage: send to output directly
	if sp.middleware == nil {
		sp.session.putPacket(DirectionOutput, packet)
		return
	}

	// Execute middleware asynchronously
	sp.middleware(sp, MediaData{
		Type:      MediaDataTypePacket,
		Packet:    packet,
		CreatedAt: time.Now(),
		Sender:    sender,
	})

	// Apply post-filter if exists
	if sp.postFilter != nil {
		if shouldSkip, err := sp.postFilter(packet); err != nil {
			sp.CauseError(sp, err)
		} else if shouldSkip {
			return
		}
	}

	// Forward to next stage asynchronously
	if sp.nextStage != nil {
		sp.nextStage.enqueueEvent(&MediaData{
			Type:      MediaDataTypePacket,
			Packet:    packet,
			CreatedAt: time.Now(),
			Sender:    sender,
		})
	}
}

// enqueueEvent adds event to the queue (non-blocking)
func (sp *PipelineStage) enqueueEvent(data *MediaData) {
	if !sp.workerRunning {
		return
	}
	select {
	case sp.eventQueue <- data:
	default:
		// Queue full, drop event (could add retry logic here)
	}
}
