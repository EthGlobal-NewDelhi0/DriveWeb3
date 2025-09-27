package sensor

import (
	"math"
	"math/rand"
	"peerdrive/app/pkg/models"
	"time"
)

// MockManager simulates sensor data for development.
type MockManager struct {
	stopChan           chan struct{}
	state              models.VehicleState
	previousPosition   models.Position // Previous position for velocity calculation
	direction          float64         // Current movement direction in radians
	speed              float64         // Current speed in units per second
	roadPath           int             // Which road path the vehicle is following (0-4)
	lastVelocityUpdate time.Time       // Time when velocity was last calculated
}

func NewMockManager(peerId string) *MockManager {
	// Initialize random position on one of the roads
	roadPositions := []models.Position{
		{Lat: 0.2, Lng: 0.5}, // On vertical road 1
		{Lat: 1.0, Lng: 0.5}, // On vertical road 2
		{Lat: 1.8, Lng: 0.5}, // On vertical road 3
		{Lat: 0.5, Lng: 0.2}, // On horizontal road 1
		{Lat: 0.5, Lng: 1.8}, // On horizontal road 2
	}

	// Pick a random starting road
	roadIndex := rand.Intn(len(roadPositions))
	startPos := roadPositions[roadIndex]

	// Random direction (0, π/2, π, 3π/2 for cardinal directions)
	directions := []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2}
	direction := directions[rand.Intn(len(directions))]

	return &MockManager{
		stopChan:           make(chan struct{}),
		direction:          direction,
		speed:              0.1 + rand.Float64()*0.2, // Speed between 0.1-0.3 units/sec
		roadPath:           roadIndex,
		previousPosition:   startPos, // Initialize previous position
		lastVelocityUpdate: time.Now(),
		state: models.VehicleState{
			Position: startPos,
			Velocity: models.Velocity{
				Lat: 0, // Initial velocity is 0
				Lng: 0,
			},
		},
	}
}

func (m *MockManager) StartTracking() (<-chan models.VehicleState, error) {
	outputChan := make(chan models.VehicleState)

	go func() {
		defer close(outputChan)
		ticker := time.NewTicker(time.Second / 10) // 10 updates per second for smooth movement
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.updatePosition()
				m.state.Timestamp = time.Now().UnixMilli()
				outputChan <- m.state
			case <-m.stopChan:
				return
			}
		}
	}()

	return outputChan, nil
}

// updatePosition simulates realistic vehicle movement on roads
func (m *MockManager) updatePosition() {
	deltaTime := 0.1 // 0.1 seconds between updates

	// Calculate movement based on current direction and speed
	deltaX := math.Cos(m.direction) * m.speed * deltaTime
	deltaY := math.Sin(m.direction) * m.speed * deltaTime

	// Update position
	newX := m.state.Position.Lat + deltaX
	newY := m.state.Position.Lng + deltaY

	// Keep vehicle within map bounds and on roads
	newX, newY = m.constrainToRoads(newX, newY)

	// Update position
	m.state.Position.Lat = newX
	m.state.Position.Lng = newY

	// Update velocity every 0.5 seconds using (current position - previous position) / 0.5
	now := time.Now()
	if now.Sub(m.lastVelocityUpdate) >= 500*time.Millisecond {
		// Calculate velocity: (coordinates - previous coordinates) / 0.5 seconds
		velocityTime := 0.25
		m.state.Velocity.Lat = (m.state.Position.Lat - m.previousPosition.Lat) / velocityTime
		m.state.Velocity.Lng = (m.state.Position.Lng - m.previousPosition.Lng) / velocityTime

		// Update previous position and timestamp
		m.previousPosition = m.state.Position
		m.lastVelocityUpdate = now
	}

	// Occasionally change direction at intersections
	if rand.Float64() < 0.02 { // 2% chance per update
		m.changeDirectionAtIntersection()
	}
}

// constrainToRoads keeps the vehicle on valid road areas
func (m *MockManager) constrainToRoads(x, y float64) (float64, float64) {
	// Define road boundaries
	verticalRoads := [][2]float64{{0.0, 0.4}, {0.8, 1.2}, {1.6, 2.0}}
	horizontalRoads := [][2]float64{{0.0, 0.4}, {1.6, 2.0}}

	// Check if position is on a road
	onVerticalRoad := false
	onHorizontalRoad := false

	for _, road := range verticalRoads {
		if x >= road[0] && x <= road[1] {
			onVerticalRoad = true
			break
		}
	}

	for _, road := range horizontalRoads {
		if y >= road[0] && y <= road[1] {
			onHorizontalRoad = true
			break
		}
	}

	// If not on any road, find closest road
	if !onVerticalRoad && !onHorizontalRoad {
		// Find closest vertical road
		minDistV := math.Inf(1)
		closestVRoad := 0
		for i, road := range verticalRoads {
			center := (road[0] + road[1]) / 2
			dist := math.Abs(x - center)
			if dist < minDistV {
				minDistV = dist
				closestVRoad = i
			}
		}

		// Find closest horizontal road
		minDistH := math.Inf(1)
		closestHRoad := 0
		for i, road := range horizontalRoads {
			center := (road[0] + road[1]) / 2
			dist := math.Abs(y - center)
			if dist < minDistH {
				minDistH = dist
				closestHRoad = i
			}
		}

		// Move to closest road
		if minDistV < minDistH {
			// Move to vertical road
			road := verticalRoads[closestVRoad]
			x = road[0] + (road[1]-road[0])/2 // Center of road
		} else {
			// Move to horizontal road
			road := horizontalRoads[closestHRoad]
			y = road[0] + (road[1]-road[0])/2 // Center of road
		}
	}

	// Keep within map bounds
	x = math.Max(0, math.Min(2.0, x))
	y = math.Max(0, math.Min(2.0, y))

	return x, y
}

// changeDirectionAtIntersection randomly changes direction when near an intersection
func (m *MockManager) changeDirectionAtIntersection() {
	// Define possible directions (North, East, South, West)
	directions := []float64{math.Pi / 2, 0, 3 * math.Pi / 2, math.Pi}

	// Pick a random new direction
	newDirection := directions[rand.Intn(len(directions))]

	// Don't reverse direction immediately
	if math.Abs(newDirection-m.direction) > math.Pi/2 && math.Abs(newDirection-m.direction) < 3*math.Pi/2 {
		m.direction = newDirection

		// Slightly randomize speed
		m.speed = 0.1 + rand.Float64()*0.2
	}
}

func (m *MockManager) StopTracking() {
	close(m.stopChan)
}
