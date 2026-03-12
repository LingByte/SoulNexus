package media

import (
	"fmt"
	"sync"
)

// RoutingStrategy defines how packets are routed
type RoutingStrategy int

const (
	// StrategyBroadcast sends to all outputs
	StrategyBroadcast RoutingStrategy = iota
	// StrategyRoundRobin distributes across outputs
	StrategyRoundRobin
	// StrategyFirstAvailable uses first available output
	StrategyFirstAvailable
)

// RouteRule defines routing rules
type RouteRule struct {
	Condition func(packet MediaPacket) bool
	Targets   []string // Transport IDs
	Strategy  RoutingStrategy
}

// Router manages packet routing
type Router struct {
	rules           []RouteRule
	defaultStrategy RoutingStrategy
	mu              sync.RWMutex
	roundRobinIndex int
}

// NewRouter creates a new router
func NewRouter(defaultStrategy RoutingStrategy) *Router {
	return &Router{
		rules:           make([]RouteRule, 0),
		defaultStrategy: defaultStrategy,
	}
}

// AddRule adds a routing rule
func (r *Router) AddRule(rule RouteRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

// Route determines where to send a packet
func (r *Router) Route(packet MediaPacket, availableTransports []*TransportConnector) []*TransportConnector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check rules in order
	for _, rule := range r.rules {
		if rule.Condition != nil && rule.Condition(packet) {
			return r.applyStrategy(rule.Strategy, rule.Targets, availableTransports)
		}
	}

	// Use default strategy
	return r.applyStrategy(r.defaultStrategy, nil, availableTransports)
}

// applyStrategy applies routing strategy
func (r *Router) applyStrategy(strategy RoutingStrategy, targets []string, available []*TransportConnector) []*TransportConnector {
	if len(available) == 0 {
		return nil
	}

	switch strategy {
	case StrategyBroadcast:
		return available

	case StrategyRoundRobin:
		if len(available) == 0 {
			return nil
		}
		r.roundRobinIndex = (r.roundRobinIndex + 1) % len(available)
		return []*TransportConnector{available[r.roundRobinIndex]}

	case StrategyFirstAvailable:
		if len(available) > 0 {
			return []*TransportConnector{available[0]}
		}
		return nil

	default:
		return available
	}
}

// TransportConnector represents a connection to a transport
type TransportConnector struct {
	ID        string
	Transport MediaTransport
	Direction string // "input" or "output"
	Active    bool
	mu        sync.RWMutex
}

// NewTransportConnector creates a new transport connector
func NewTransportConnector(id string, transport MediaTransport, direction string) *TransportConnector {
	return &TransportConnector{
		ID:        id,
		Transport: transport,
		Direction: direction,
		Active:    true,
	}
}

// String returns string representation
func (tc *TransportConnector) String() string {
	return fmt.Sprintf("TransportConnector{ID: %s, Direction: %s, Active: %v}", tc.ID, tc.Direction, tc.Active)
}

// SetActive sets the active state
func (tc *TransportConnector) SetActive(active bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.Active = active
}

// IsActive checks if connector is active
func (tc *TransportConnector) IsActive() bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.Active
}
