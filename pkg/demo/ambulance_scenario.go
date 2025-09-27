package demo

import (
	"fmt"
	"sync"
	"time"

	"peerdrive/app/pkg/models"
)

// ScenarioType defines the type of scenario being run
type ScenarioType int

const (
	NoV2V ScenarioType = iota
	WithV2V
)

// ScenarioResult holds the results of a completed scenario
type ScenarioResult struct {
	ScenarioType    ScenarioType
	AmbulanceTime   time.Duration
	NormalCarTime   time.Duration
	PatientOutcome  string
	V2VAlertSent    bool
	CarYieldedSpace bool
	AverageAmbSpeed float64
}

// AmbulanceScenario manages the ambulance priority demonstration
type AmbulanceScenario struct {
	vehicles     []*DemoVehicle
	scenarioType ScenarioType
	isRunning    bool
	startTime    time.Time
	result       *ScenarioResult
	mu           sync.RWMutex
	stopChan     chan struct{}
	updateTicker *time.Ticker
	v2vAlertSent bool
	carYielded   bool
}

// NewAmbulanceScenario creates a new ambulance priority scenario
func NewAmbulanceScenario() *AmbulanceScenario {
	return &AmbulanceScenario{
		vehicles: make([]*DemoVehicle, 0),
		stopChan: make(chan struct{}),
	}
}

// StartScenario begins running the specified scenario type
func (s *AmbulanceScenario) StartScenario(scenarioType ScenarioType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scenario already running")
	}

	s.scenarioType = scenarioType
	s.isRunning = true
	s.startTime = time.Now()
	s.result = nil
	s.v2vAlertSent = false
	s.carYielded = false

	// Initialize vehicles
	s.initializeVehicles()

	// Start simulation loop
	s.updateTicker = time.NewTicker(50 * time.Millisecond) // 20 FPS
	go s.runSimulation()

	return nil
}

// StopScenario stops the current scenario and returns results
func (s *AmbulanceScenario) StopScenario() *ScenarioResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return s.result
	}

	s.isRunning = false
	if s.updateTicker != nil {
		s.updateTicker.Stop()
	}
	// Only close channel if it's still open
	select {
	case <-s.stopChan:
		// Channel already closed
	default:
		close(s.stopChan)
	}

	// Calculate final results
	s.calculateResults()
	return s.result
}

// IsRunning returns whether a scenario is currently active
func (s *AmbulanceScenario) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// GetCurrentState returns the current vehicle states for visualization
func (s *AmbulanceScenario) GetCurrentState() models.Ledger {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ledger := models.Ledger{
		Ledger: make(map[string]models.NeighbourEntry),
	}

	for _, vehicle := range s.vehicles {
		vehicleState := vehicle.ToVehicleState()
		ledger.Ledger[vehicle.ID] = models.NeighbourEntry{
			Neighbour:    1,
			VehicleState: vehicleState,
		}
		// Debug: uncomment to see vehicle positions
		// fmt.Printf("Demo vehicle %s at position (%.2f, %.2f), ambulance: %v\n",
		//	vehicle.ID, vehicleState.Position.Lat, vehicleState.Position.Lng, vehicle.IsAmbulance)
	}

	return ledger
}

// GetScenarioStatus returns current scenario information
func (s *AmbulanceScenario) GetScenarioStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning {
		if s.result != nil {
			return fmt.Sprintf("Completed: %s", s.formatResult(s.result))
		}
		return "No scenario running"
	}

	elapsed := time.Since(s.startTime)
	scenarioName := "Without V2V"
	if s.scenarioType == WithV2V {
		scenarioName = "With V2V"
	}

	status := fmt.Sprintf("Running: %s (%.1fs)", scenarioName, elapsed.Seconds())

	if s.v2vAlertSent {
		status += " - V2V Alert Sent!"
	}
	if s.carYielded {
		status += " - Car Yielding Space"
	}

	return status
}

// initializeVehicles sets up the initial vehicle positions
func (s *AmbulanceScenario) initializeVehicles() {
	// Create vehicles positioned before the intersection
	// Normal car starts closer to intersection, ambulance starts further back
	s.vehicles = []*DemoVehicle{
		NewDemoVehicle("NORMAL_CAR", false, 0.7), // Near intersection approach
		NewDemoVehicle("AMBULANCE", true, 0.3),   // Further back
	}

	// Set initial movement states
	for _, vehicle := range s.vehicles {
		// Both vehicles start on horizontal road moving right toward intersection
		vehicle.CurrentRoad = "horizontal_bottom" // Bottom horizontal road
		vehicle.TargetIntersection = "center"     // Both heading to center intersection
		vehicle.Velocity = 0.15                   // Moderate speed for visibility
	}
}

// runSimulation runs the main simulation loop
func (s *AmbulanceScenario) runSimulation() {
	defer s.updateTicker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-s.updateTicker.C:
			s.updateSimulation()
		}
	}
}

// updateSimulation updates the simulation state
func (s *AmbulanceScenario) updateSimulation() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	dt := 0.05 // 50ms timestep

	// Find vehicles
	var normalCar, ambulance *DemoVehicle
	for _, vehicle := range s.vehicles {
		if vehicle.IsAmbulance {
			ambulance = vehicle
		} else {
			normalCar = vehicle
		}
	}

	// Check if ambulance is approaching normal car
	if s.scenarioType == WithV2V && !s.v2vAlertSent {
		if ambulance != nil && normalCar != nil {
			distance := normalCar.Position - ambulance.Position
			if distance > 0 && distance < 0.5 { // Within 50m
				s.v2vAlertSent = true
			}
		}
	}

	// Update vehicle behaviors
	if normalCar != nil {
		s.updateNormalCar(normalCar, ambulance, dt)
	}

	if ambulance != nil {
		s.updateAmbulance(ambulance, normalCar, dt)
	}

	// Check for completion
	s.checkCompletion()
}

// updateNormalCar updates the normal car behavior
func (s *AmbulanceScenario) updateNormalCar(car *DemoVehicle, ambulance *DemoVehicle, dt float64) {
	if car.HasFinished {
		return
	}

	// Check if approaching intersection and V2V alert received
	if s.scenarioType == WithV2V && s.v2vAlertSent && !car.YieldedToAmbulance {
		if car.CurrentRoad == "horizontal_bottom" && car.Position >= 0.8 {
			// Car is approaching intersection, yield to ambulance
			s.implementV2VYielding(car, ambulance)
			return
		}
	}

	// Normal movement logic
	s.updateVehicleMovement(car, dt)
}

// updateAmbulance updates the ambulance behavior
func (s *AmbulanceScenario) updateAmbulance(ambulance *DemoVehicle, normalCar *DemoVehicle, dt float64) {
	if ambulance.HasFinished {
		return
	}

	// Ambulance behavior based on scenario type
	if s.scenarioType == NoV2V {
		// Without V2V: get stuck behind normal car at intersection
		if normalCar != nil && !normalCar.HasFinished {
			// Check if both vehicles are on same road and ambulance is behind
			if normalCar.CurrentRoad == ambulance.CurrentRoad {
				distance := normalCar.Position - ambulance.Position
				if distance < 0.2 && distance > 0 {
					ambulance.Velocity = normalCar.Velocity // Match car speed (stuck behind)
				}
			}
		}
	} else {
		// With V2V: can move faster when car yields
		if s.carYielded {
			ambulance.Velocity = 0.25 // Faster speed for ambulance
		}
	}

	// Use general movement logic
	s.updateVehicleMovement(ambulance, dt)
}

// checkCompletion checks if scenario is complete
func (s *AmbulanceScenario) checkCompletion() {
	allFinished := true
	for _, vehicle := range s.vehicles {
		if !vehicle.HasFinished {
			allFinished = false
			break
		}
	}

	if allFinished {
		s.isRunning = false
		s.calculateResults()
	}
}

// calculateResults computes the final scenario results
func (s *AmbulanceScenario) calculateResults() {
	var ambulance, normalCar *DemoVehicle
	for _, vehicle := range s.vehicles {
		if vehicle.IsAmbulance {
			ambulance = vehicle
		} else {
			normalCar = vehicle
		}
	}

	if ambulance == nil {
		return
	}

	ambulanceTime := ambulance.GetJourneyTime()
	normalCarTime := time.Duration(0)
	if normalCar != nil {
		normalCarTime = normalCar.GetJourneyTime()
	}

	// Calculate average speed
	avgSpeed := 2.0 / ambulanceTime.Seconds() // distance / time

	// Determine patient outcome
	patientOutcome := "❌ Patient condition critical"
	if ambulanceTime.Seconds() < 25.0 { // Threshold for saving patient
		patientOutcome = "✅ Patient saved!"
	}

	s.result = &ScenarioResult{
		ScenarioType:    s.scenarioType,
		AmbulanceTime:   ambulanceTime,
		NormalCarTime:   normalCarTime,
		PatientOutcome:  patientOutcome,
		V2VAlertSent:    s.v2vAlertSent,
		CarYieldedSpace: s.carYielded,
		AverageAmbSpeed: avgSpeed,
	}
}

// formatResult formats a result for display
// implementV2VYielding handles the car yielding behavior with V2V
func (s *AmbulanceScenario) implementV2VYielding(car *DemoVehicle, ambulance *DemoVehicle) {
	// Car recognizes ambulance behind and implements yielding strategy
	// Car will bypass the intersection and come back later
	car.YieldedToAmbulance = true
	car.Velocity = 0.2 // Maintain reasonable speed to bypass intersection
	s.carYielded = true

	// Change target to bypass the intersection
	car.TargetIntersection = "bypass"
}

// updateVehicleMovement handles general vehicle movement on roads
func (s *AmbulanceScenario) updateVehicleMovement(vehicle *DemoVehicle, dt float64) {
	switch vehicle.CurrentRoad {
	case "horizontal_bottom":
		// Moving horizontally toward intersection
		vehicle.Position += vehicle.Velocity * dt

		// Check behavior based on target
		if vehicle.TargetIntersection == "center" && vehicle.Position >= 1.0 {
			// Normal behavior: turn into intersection
			vehicle.CurrentRoad = "vertical_center"
			vehicle.Position = 0.2 // Start at bottom of vertical road
			vehicle.TargetIntersection = "complete"
		} else if vehicle.TargetIntersection == "bypass" && vehicle.Position >= 1.4 {
			// Bypass behavior: continue straight past intersection
			vehicle.CurrentRoad = "horizontal_bypass"
			vehicle.Position = 1.4 // Continue on horizontal road
			vehicle.TargetIntersection = "u_turn"
		}

	case "horizontal_bypass":
		// Continue straight past the intersection
		vehicle.Position += vehicle.Velocity * dt

		// When far enough, start U-turn
		if vehicle.Position >= 1.8 {
			vehicle.CurrentRoad = "u_turn"
			vehicle.Position = 1.8
			vehicle.TargetIntersection = "return"
			vehicle.Velocity = 0.15 // Slow down for U-turn
		}

	case "u_turn":
		// Making U-turn (simulated as stationary for a moment)
		vehicle.Position += vehicle.Velocity * dt * 0.5 // Very slow movement during turn

		if vehicle.Position >= 1.9 {
			// Complete U-turn, now heading back
			vehicle.CurrentRoad = "horizontal_return"
			vehicle.Position = 1.8 // Start heading back
			vehicle.TargetIntersection = "center_delayed"
			vehicle.Velocity = 0.2 // Resume normal speed
		}

	case "horizontal_return":
		// Heading back toward intersection
		vehicle.Position -= vehicle.Velocity * dt // Moving in opposite direction

		// Check if ambulance has passed through intersection
		if vehicle.Position <= 1.0 && s.ambulanceHasPassedIntersection() {
			// Now safe to take the intersection
			vehicle.CurrentRoad = "vertical_center"
			vehicle.Position = 0.2
			vehicle.TargetIntersection = "complete"
		}

	case "vertical_center":
		// Moving vertically through intersection
		vehicle.Position += vehicle.Velocity * dt

		// Check if completed the turn
		if vehicle.Position >= 1.6 {
			vehicle.HasFinished = true
			now := time.Now()
			vehicle.FinishTime = &now
		}
	}
}

// ambulanceHasPassedIntersection checks if the ambulance has completed its turn
func (s *AmbulanceScenario) ambulanceHasPassedIntersection() bool {
	for _, vehicle := range s.vehicles {
		if vehicle.IsAmbulance {
			return vehicle.CurrentRoad == "vertical_center" && vehicle.Position > 1.0
		}
	}
	return false
}

func (s *AmbulanceScenario) formatResult(result *ScenarioResult) string {
	scenarioName := "No V2V"
	if result.ScenarioType == WithV2V {
		scenarioName = "With V2V"
	}

	return fmt.Sprintf("%s: Ambulance time %.1fs - %s",
		scenarioName,
		result.AmbulanceTime.Seconds(),
		result.PatientOutcome)
}
