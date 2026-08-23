package main

import (
	"context"
	"sync"
)

type jobManager struct {
	mu     sync.Mutex
	cancel map[string]context.CancelFunc
}

func newJobManager() *jobManager {
	return &jobManager{cancel: make(map[string]context.CancelFunc)}
}

func (m *jobManager) start(parent context.Context, filename string) context.Context {
	ctx, cancel := context.WithCancel(parent)
	m.mu.Lock()
	if previous, ok := m.cancel[filename]; ok {
		previous()
	}
	m.cancel[filename] = cancel
	m.mu.Unlock()
	return ctx
}

func (m *jobManager) finish(filename string) {
	m.mu.Lock()
	delete(m.cancel, filename)
	m.mu.Unlock()
}

func (m *jobManager) cancelJob(filename string) bool {
	m.mu.Lock()
	cancel, ok := m.cancel[filename]
	m.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}

var jobs = newJobManager()
