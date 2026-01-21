package storage

import (
	"encoding/json"
	"fmt"
	"time"
)

// StateSyncMessage represents a state synchronization message
type StateSyncMessage struct {
	Type      string        `json:"type"`      // "state_sync", "state_request"
	State     *ClusterState `json:"state"`     // Full state for sync
	Timestamp time.Time     `json:"timestamp"` // Message timestamp
	NodeName  string        `json:"node_name"` // Originating node
}

// BroadcastState broadcasts the current state to all nodes
func (s *Storage) BroadcastState(broadcast func([]byte) error, nodeName string) error {
	s.mu.RLock()
	state := s.GetState()
	s.mu.RUnlock()

	msg := StateSyncMessage{
		Type:      "state_sync",
		State:     state,
		Timestamp: time.Now(),
		NodeName:  nodeName,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal state sync message: %w", err)
	}

	return broadcast(data)
}

// HandleStateSyncMessage handles incoming state synchronization messages
func (s *Storage) HandleStateSyncMessage(data []byte) error {
	var msg StateSyncMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal state sync message: %w", err)
	}

	switch msg.Type {
	case "state_sync":
		// Merge incoming state
		if msg.State != nil {
			return s.MergeState(msg.State)
		}
	case "state_request":
		// Another node is requesting our state
		// This would trigger a BroadcastState call
		s.logger.Debugf("State request received from node: %s", msg.NodeName)
	}

	return nil
}

// StartPeriodicBackup starts periodic backup routine
func (s *Storage) StartPeriodicBackup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.Backup(); err != nil {
				s.logger.Errorf("Failed to create backup: %v", err)
			}
		}
	}()
}

// StartPeriodicSync starts periodic state synchronization
func (s *Storage) StartPeriodicSync(interval time.Duration, broadcast func([]byte) error, nodeName string) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.BroadcastState(broadcast, nodeName); err != nil {
				s.logger.Debugf("Failed to broadcast state: %v", err)
			}
		}
	}()
}

// RequestState broadcasts a state request to all nodes
func (s *Storage) RequestState(broadcast func([]byte) error, nodeName string) error {
	msg := StateSyncMessage{
		Type:      "state_request",
		Timestamp: time.Now(),
		NodeName:  nodeName,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal state request: %w", err)
	}

	return broadcast(data)
}
