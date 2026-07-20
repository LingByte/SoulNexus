package models

import (
	"context"
	"sync"
)

type memoryReserveBackend struct {
	mu     sync.Mutex
	byCall map[string]memoryReservation
	held   map[uint]int64
}

type memoryReservation struct {
	tenantID uint
	minutes  int64
}

func newMemoryReserveBackend() *memoryReserveBackend {
	return &memoryReserveBackend{
		byCall: make(map[string]memoryReservation),
		held:   make(map[uint]int64),
	}
}

func (m *memoryReserveBackend) lockTenant(_ context.Context, _ uint) (func(), error) {
	m.mu.Lock()
	return m.mu.Unlock, nil
}

func (m *memoryReserveBackend) heldMinutes(_ context.Context, tenantID uint) (int64, error) {
	return m.held[tenantID], nil
}

func (m *memoryReserveBackend) putReservation(_ context.Context, tenantID uint, callID string, minutes int64) error {
	if _, ok := m.byCall[callID]; ok {
		return nil
	}
	m.byCall[callID] = memoryReservation{tenantID: tenantID, minutes: minutes}
	m.held[tenantID] += minutes
	return nil
}

func (m *memoryReserveBackend) dropReservation(_ context.Context, callID string) error {
	res, ok := m.byCall[callID]
	if !ok {
		return nil
	}
	delete(m.byCall, callID)
	if m.held[res.tenantID] >= res.minutes {
		m.held[res.tenantID] -= res.minutes
	} else {
		m.held[res.tenantID] = 0
	}
	return nil
}

func (m *memoryReserveBackend) setBalanceHint(_ context.Context, _ uint, _ int64) error {
	return nil
}

var _ reserveBackend = (*memoryReserveBackend)(nil)
