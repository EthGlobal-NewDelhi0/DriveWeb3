package demo

import (
	"time"

	"peerdrive/app/pkg/models"
)

// DemoVehicle represents a vehicle in the ambulance demo
type DemoVehicle struct {
	ID                 string
	IsAmbulance        bool
	Position           float64 // Position along current road (0-2.0)
	Velocity           float64 // Current velocity in m/s
	MaxVelocity        float64 // Maximum allowed velocity
	StartTime          time.Time
	FinishTime         *time.Time
	HasFinished        bool
	Lane               int    // 0 = main lane, 1 = shoulder/side
	CurrentRoad        string // "horizontal_bottom", "vertical_center", etc.
	TargetIntersection string // "center", "complete", etc.
	YieldedToAmbulance bool   // Whether car has yielded space
}

// VehicleBehavior defines how vehicles behave in different scenarios
type VehicleBehavior interface {
	Update(dt float64, vehicles []*DemoVehicle, hasV2V bool) // Update vehicle state
	ShouldYield(ambulance *DemoVehicle, hasV2V bool) bool    // Check if should yield to ambulance
}

// NormalCarBehavior implements behavior for regular vehicles
type NormalCarBehavior struct{}

func (b *NormalCarBehavior) Update(dt float64, vehicles []*DemoVehicle, hasV2V bool) {
	// Implementation will be added in scenario logic
}

func (b *NormalCarBehavior) ShouldYield(ambulance *DemoVehicle, hasV2V bool) bool {
	if !hasV2V {
		return false // No communication, can't detect ambulance
	}

	// With V2V, yield if ambulance is within 50m behind
	return ambulance != nil && ambulance.IsAmbulance
}

// AmbulanceBehavior implements behavior for emergency vehicles
type AmbulanceBehavior struct{}

func (b *AmbulanceBehavior) Update(dt float64, vehicles []*DemoVehicle, hasV2V bool) {
	// Implementation will be added in scenario logic
}

func (b *AmbulanceBehavior) ShouldYield(ambulance *DemoVehicle, hasV2V bool) bool {
	return false // Ambulances don't yield
}

// NewDemoVehicle creates a new demo vehicle
func NewDemoVehicle(id string, isAmbulance bool, startPos float64) *DemoVehicle {
	maxVel := 0.3 // Scaled down for demo visibility
	if isAmbulance {
		maxVel = 0.5 // Faster for ambulance
	}

	return &DemoVehicle{
		ID:          id,
		IsAmbulance: isAmbulance,
		Position:    startPos,
		Velocity:    0.0,
		MaxVelocity: maxVel,
		StartTime:   time.Now(),
		HasFinished: false,
		Lane:        0, // Start in main lane
	}
}

// ToVehicleState converts DemoVehicle to models.VehicleState for visualization
func (v *DemoVehicle) ToVehicleState() models.VehicleState {
	var lat, lng float64

	// Map vehicle position based on current road
	switch v.CurrentRoad {
	case "horizontal_bottom":
		// Moving horizontally on bottom road (y=0 to y=0.4, center at y=0.2)
		lat = v.Position
		lng = 0.2

	case "horizontal_bypass":
		// Continuing straight past intersection
		lat = v.Position
		lng = 0.3 // Slightly higher to show bypassing

	case "u_turn":
		// Making U-turn - show at far end
		lat = v.Position
		lng = 0.4 // Even higher to show U-turn

	case "horizontal_return":
		// Returning back to intersection
		lat = v.Position
		lng = 0.1 // Lower lane to show return direction

	case "vertical_center":
		// Moving vertically on center road (x=0.8 to x=1.2, center at x=1.0)
		lat = 1.0
		lng = v.Position

	default:
		// Fallback to horizontal bottom road
		lat = v.Position
		lng = 0.2
	}

	priority := 1
	if v.IsAmbulance {
		priority = 10 // High priority for emergency vehicles
	}

	// Calculate velocity direction based on current road
	var velLat, velLng float64

	switch v.CurrentRoad {
	case "horizontal_bottom", "horizontal_bypass", "horizontal_return":
		// Moving horizontally
		if v.CurrentRoad == "horizontal_return" {
			velLat = -v.Velocity / 10.0 // Moving left (negative direction)
		} else {
			velLat = v.Velocity / 10.0 // Moving right (positive direction)
		}
		velLng = 0.0

	case "u_turn":
		// During U-turn, show minimal movement
		velLat = 0.0
		velLng = 0.0

	case "vertical_center":
		// Moving vertically (upward/northward)
		velLat = 0.0
		velLng = v.Velocity / 10.0 // Moving up (positive direction)

	default:
		// Default to horizontal movement
		velLat = v.Velocity / 10.0
		velLng = 0.0
	}

	return models.VehicleState{
		Position: models.Position{
			Lat:      lat,
			Lng:      lng,
			Accuracy: 0.0,
		},
		Velocity: models.Velocity{
			Lat: velLat,
			Lng: velLng,
		},
		Acceleration: models.Acceleration{
			Lat: 0.0,
			Lng: 0.0,
		},
		Timestamp: time.Now().UnixMilli(),
		Priority:  priority,
	}
}

// IsAtDestination checks if vehicle has reached the end
func (v *DemoVehicle) IsAtDestination() bool {
	return v.Position >= 2.0 // End of road
}

// GetJourneyTime returns the time taken to complete journey
func (v *DemoVehicle) GetJourneyTime() time.Duration {
	if v.FinishTime != nil {
		return v.FinishTime.Sub(v.StartTime)
	}
	return time.Since(v.StartTime)
}
