package utils

// Signals
type SignalHandler func(sender any, params ...any)

type SigHandler struct {
	ID      uint
	Handler SignalHandler
}

const (
	evTypeAdd = iota
	evEypeDel
)

type SigHandlerEvent struct {
	EvType     int
	SignalName string
	SigHandler SigHandler
}

type Signals struct {
	lastID      uint
	sigHandlers map[string][]SigHandler
	inLoop      bool
	events      []SigHandlerEvent
}

var sig *Signals

func init() {
	Sig()
}

func Sig() *Signals {
	if sig == nil {
		sig = NewSignals()
	}
	return sig
}

func NewSignals() *Signals {
	return &Signals{
		lastID:      0,
		sigHandlers: map[string][]SigHandler{},
		inLoop:      false,
		events:      []SigHandlerEvent{},
	}
}

func (s *Signals) processEvents() {
	if len(s.events) <= 0 || s.inLoop {
		return
	}
	defer func() {
		s.events = nil
	}()

	for _, v := range s.events {
		sigs, ok := s.sigHandlers[v.SignalName]
		if !ok {
			sigs = make([]SigHandler, 0)
		}
		switch v.EvType {
		case evTypeAdd:
			sigs = append(sigs, v.SigHandler)
		case evEypeDel:
			for i := 0; i < len(sigs); i++ {
				if sigs[i].ID == v.SigHandler.ID {
					sigs = append(sigs[0:i], sigs[i+1:]...)
					break
				}
			}
		}
		s.sigHandlers[v.SignalName] = sigs
	}
}

func (s *Signals) Connect(event string, handler SignalHandler) uint {
	s.lastID += 1
	ev := SigHandlerEvent{
		EvType:     evTypeAdd,
		SignalName: event,
		SigHandler: SigHandler{
			ID:      s.lastID,
			Handler: handler,
		},
	}
	s.events = append(s.events, ev)
	s.processEvents()
	return s.lastID
}

func (s *Signals) Disconnect(event string, id uint) {
	ev := SigHandlerEvent{
		EvType:     evEypeDel,
		SignalName: event,
		SigHandler: SigHandler{
			ID: id,
		},
	}
	s.events = append(s.events, ev)
	s.processEvents()
}

func (s *Signals) Clear(events ...string) {
	for _, event := range events {
		delete(s.sigHandlers, event)
	}
}

func (s *Signals) Emit(event string, sender any, params ...any) {
	s.inLoop = true
	defer func() {
		s.inLoop = false
		s.processEvents()
	}()

	sigs, ok := s.sigHandlers[event]
	if !ok {
		return
	}
	for _, sig := range sigs {
		sig.Handler(sender, params...)
	}
}
