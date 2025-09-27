package models

import "sync"

// Note: In Go, we use struct tags for serialization (e.g., to JSON)
type Position struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Accuracy float64 `json:"accuracy"`
}

type Velocity struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type Acceleration struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type VehicleState struct {
	Position     Position     `json:"position"`
	Velocity     Velocity     `json:"velocity"`
	Acceleration Acceleration `json:"acceleration"`
	Timestamp    int64        `json:"timestamp"`
	Priority     int          `json:"priority" validate:"min=1,max=10"`
}

type NeighbourEntry struct {
	Neighbour    int64        `json:"neighbour"`
	VehicleState VehicleState `json:"vehicleState"`
}

// Ledger is a map from peer ID (string) to NeighbourEntry for O(1) lookup.
type Ledger struct {
	Ledger map[string]NeighbourEntry `json:"ledger"`
}

// NetworkState manages the shared state of the network.
// It includes a RWMutex to handle concurrent reads and writes safely.
type NetworkState struct {
	mu                   sync.RWMutex
	Peers                map[string]VehicleState // peerId -> VehicleState
	LocalState           VehicleState
	EmergencyAlerts      []string // Simplified for now
	TrafficRecommendations string   // Simplified for now
}