package main

import (
	"encoding/json"
	"sync"
)

type StreamEvent struct {
	Type     string      `json:"type"`
	Filename string      `json:"filename,omitempty"`
	DeviceID string      `json:"deviceId,omitempty"`
	Status   string      `json:"status,omitempty"`
	Progress float64     `json:"progress,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

type eventHub struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subscribers: make(map[chan []byte]struct{})}
}

func (h *eventHub) subscribe() chan []byte {
	channel := make(chan []byte, 16)
	h.mu.Lock()
	h.subscribers[channel] = struct{}{}
	h.mu.Unlock()
	return channel
}

func (h *eventHub) unsubscribe(channel chan []byte) {
	h.mu.Lock()
	if _, ok := h.subscribers[channel]; ok {
		delete(h.subscribers, channel)
		close(channel)
	}
	h.mu.Unlock()
}

func (h *eventHub) publish(event StreamEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for channel := range h.subscribers {
		select {
		case channel <- data:
		default:
		}
	}
}

var events = newEventHub()
