package strategy

import "peerdrive/app/pkg/models"

type Engine struct {
	state *models.NetworkState
}

func NewEngine(state *models.NetworkState) *Engine {
	return &Engine{state: state}
}

func (e *Engine) Run() {
	// This can be a loop that periodically runs calculations
	// e.state.mu.RLock() // Use a read-lock for safety
	// ... perform calculations on e.state.Peers ...
	// e.state.mu.RUnlock()
}