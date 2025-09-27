package widgets

import (
	"image/color"
	"math"

	"peerdrive/app/pkg/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	// Map grid specifications
	GridSize      = 2.0 // 2x2 grid
	RoadWidth     = 0.4 // Each road is 0.4 units wide
	NumVertical   = 2   // 2 vertical roads (third removed for green space)
	NumHorizontal = 2   // 2 horizontal roads

	// Visual constants
	MapSize = 400.0 // Display size in pixels
)

// RoadMapWidget represents a custom widget for displaying the road grid
type RoadMapWidget struct {
	widget.BaseWidget
	roads    []fyne.CanvasObject
	vehicles map[string]*VehicleMarker
	mapSize  float32
}

// VehicleMarker represents a vehicle on the map
type VehicleMarker struct {
	id        string
	carParts  []*canvas.Rectangle // Multiple rectangles forming a car shape
	position  models.Position
	velocity  models.Velocity
	direction string // "up", "down", "left", "right"
}

// NewRoadMapWidget creates a new road map widget
func NewRoadMapWidget() *RoadMapWidget {
	w := &RoadMapWidget{
		mapSize:  MapSize,
		vehicles: make(map[string]*VehicleMarker),
	}
	w.ExtendBaseWidget(w)
	w.createRoadGrid()
	return w
}

// createRoadGrid generates the road layout with exact specifications
func (w *RoadMapWidget) createRoadGrid() {
	w.roads = []fyne.CanvasObject{}

	// Colors
	backgroundColor := color.RGBA{R: 34, G: 139, B: 34, A: 255} // Forest green background
	roadColor := color.RGBA{R: 60, G: 60, B: 60, A: 255}        // Dark gray for roads
	laneColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}     // White for lane dividers

	// Create green background rectangle first
	background := canvas.NewRectangle(backgroundColor)
	background.Move(fyne.NewPos(0, 0))
	background.Resize(fyne.NewSize(w.mapSize, w.mapSize))
	w.roads = append(w.roads, background)

	// Vertical roads positions (modified to exclude third road):
	// Road 1: x=0 to x=0.4
	// Road 2: x=0.8 to x=1.2
	// Road 3: REMOVED - now covered by green patch
	verticalRoadStarts := []float64{0.0, 0.8}

	// Create vertical roads
	for _, startX := range verticalRoadStarts {
		// Convert world coordinates to pixels
		startPixelX := w.worldToPixel(startX, 0).X
		endPixelX := w.worldToPixel(startX+RoadWidth, 0).X
		roadPixelWidth := endPixelX - startPixelX

		// Create road rectangle
		road := canvas.NewRectangle(roadColor)
		road.Move(fyne.NewPos(startPixelX, 0))
		road.Resize(fyne.NewSize(roadPixelWidth, w.mapSize))
		w.roads = append(w.roads, road)

		// Add dotted lane divider in the center of the road
		// centerX := startPixelX + roadPixelWidth/2
		// w.createDottedLine(centerX, 0, centerX, w.mapSize, true, laneColor)
	}

	// Horizontal roads positions (exact as specified):
	// Road 1: y=0 to y=0.4
	// Road 2: y=1.6 to y=2.0
	horizontalRoadStarts := []float64{0.0, 1.6}

	// Create horizontal roads
	for _, startY := range horizontalRoadStarts {
		// Convert world coordinates to pixels
		startPixelY := w.worldToPixel(0, startY).Y
		endPixelY := w.worldToPixel(0, startY+RoadWidth).Y
		roadPixelHeight := startPixelY - endPixelY // Note: Y is flipped

		// Create road rectangle
		road := canvas.NewRectangle(roadColor)
		road.Move(fyne.NewPos(0, endPixelY))
		road.Resize(fyne.NewSize(w.mapSize, roadPixelHeight))
		w.roads = append(w.roads, road)

		// Add dotted lane divider in the center of the road
		centerY := endPixelY + roadPixelHeight/2
		w.createDottedLine(0, centerY, w.mapSize, centerY, false, laneColor)
	}
}

// createDottedLine creates a dotted line using small rectangles
func (w *RoadMapWidget) createDottedLine(x1, y1, x2, y2 float32, isVertical bool, lineColor color.Color) {
	dashLength := float32(15) // Length of each dash (increased for visibility)
	gapLength := float32(10)  // Length of gap between dashes (increased)
	dashWidth := float32(3)   // Width of the dash line (increased)

	if isVertical {
		// Vertical dotted line
		totalLength := y2 - y1
		numDashes := int(totalLength / (dashLength + gapLength))

		for i := 0; i < numDashes; i++ {
			dashY := y1 + float32(i)*(dashLength+gapLength)
			dash := canvas.NewRectangle(lineColor)
			dash.Move(fyne.NewPos(x1-dashWidth/2, dashY))
			dash.Resize(fyne.NewSize(dashWidth, dashLength))
			w.roads = append(w.roads, dash)
		}
	} else {
		// Horizontal dotted line
		totalLength := x2 - x1
		numDashes := int(totalLength / (dashLength + gapLength))

		for i := 0; i < numDashes; i++ {
			dashX := x1 + float32(i)*(dashLength+gapLength)
			dash := canvas.NewRectangle(lineColor)
			dash.Move(fyne.NewPos(dashX, y1-dashWidth/2))
			dash.Resize(fyne.NewSize(dashLength, dashWidth))
			w.roads = append(w.roads, dash)
		}
	}
}

// worldToPixel converts world coordinates (0-2 range) to pixel coordinates
func (w *RoadMapWidget) worldToPixel(worldX, worldY float64) fyne.Position {
	// Scale world coordinates to pixel coordinates
	pixelX := float32(worldX) * (w.mapSize / GridSize)
	pixelY := float32(worldY) * (w.mapSize / GridSize)

	// Flip Y coordinate (screen coordinates have Y=0 at top)
	pixelY = w.mapSize - pixelY

	return fyne.NewPos(pixelX, pixelY)
}

// pixelToWorld converts pixel coordinates to world coordinates (0-2 range)
func (w *RoadMapWidget) pixelToWorld(pixelX, pixelY float32) (float64, float64) {
	// Convert pixel coordinates to world coordinates
	worldX := float64(pixelX) * (GridSize / float64(w.mapSize))
	worldY := float64(w.mapSize-pixelY) * (GridSize / float64(w.mapSize))

	return worldX, worldY
}

// CreateRenderer creates the renderer for this widget
func (w *RoadMapWidget) CreateRenderer() fyne.WidgetRenderer {
	return &roadMapRenderer{
		widget:    w,
		container: container.NewWithoutLayout(),
	}
}

// roadMapRenderer implements fyne.WidgetRenderer
type roadMapRenderer struct {
	widget    *RoadMapWidget
	container *fyne.Container
}

func (r *roadMapRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)

	// Update map size if needed
	if size.Width != r.widget.mapSize || size.Height != r.widget.mapSize {
		// Use the smaller dimension to maintain square aspect ratio
		newSize := float32(math.Min(float64(size.Width), float64(size.Height)))
		r.widget.mapSize = newSize
		r.widget.createRoadGrid()
		r.Refresh()
	}
}

func (r *roadMapRenderer) MinSize() fyne.Size {
	return fyne.NewSize(300, 300)
}

func (r *roadMapRenderer) Refresh() {
	// Clear container
	r.container.RemoveAll()

	// Add roads (background layer)
	for _, road := range r.widget.roads {
		r.container.Add(road)
	}

	// Add vehicles (foreground layer)
	for _, vehicle := range r.widget.vehicles {
		for _, part := range vehicle.carParts {
			r.container.Add(part)
		}
	}

	r.container.Refresh()
}

func (r *roadMapRenderer) BackgroundColor() color.Color {
	// Background is now drawn explicitly as a rectangle
	return color.Transparent
}

func (r *roadMapRenderer) Objects() []fyne.CanvasObject {
	return r.container.Objects
}

func (r *roadMapRenderer) Destroy() {}

// GetWorldBounds returns the world coordinate bounds
func (w *RoadMapWidget) GetWorldBounds() (minX, minY, maxX, maxY float64) {
	return 0, 0, GridSize, GridSize
}

// UpdateVehicles updates all vehicle positions from the ledger
func (w *RoadMapWidget) UpdateVehicles(ledger models.Ledger) {
	// Remove vehicles no longer in ledger
	for id := range w.vehicles {
		if _, exists := ledger.Ledger[id]; !exists {
			w.removeVehicle(id)
		}
	}

	// Add or update vehicles from ledger
	for peerID, entry := range ledger.Ledger {
		pos := entry.VehicleState.Position
		vel := entry.VehicleState.Velocity
		isEmergency := entry.VehicleState.Priority >= 10 // Priority 10+ indicates emergency vehicle
		w.updateVehicle(peerID, pos.Lat, pos.Lng, vel, isEmergency)
	}

	// Refresh the display on main UI thread
	fyne.DoAndWait(func() {
		w.Refresh()
	})
}

// updateVehicle adds or updates a vehicle marker at world coordinates
func (w *RoadMapWidget) updateVehicle(id string, worldX, worldY float64, velocity models.Velocity, isEmergency bool) {
	pixel := w.worldToPixel(worldX, worldY)

	// Create new vehicle if it doesn't exist
	if _, exists := w.vehicles[id]; !exists {
		vehicleColor := w.getVehicleColor(id, isEmergency)
		carParts := w.createCarShape(vehicleColor, isEmergency)

		w.vehicles[id] = &VehicleMarker{
			id:       id,
			carParts: carParts,
			position: models.Position{
				Lat: worldX,
				Lng: worldY,
			},
			velocity:  velocity,
			direction: "right", // default direction
		}
	} else {
		// Update color if emergency status changed
		vehicle := w.vehicles[id]
		newColor := w.getVehicleColor(id, isEmergency)
		for _, part := range vehicle.carParts {
			part.FillColor = newColor
		}
	}

	// Update position and velocity
	vehicle := w.vehicles[id]
	vehicle.position.Lat = worldX
	vehicle.position.Lng = worldY
	vehicle.velocity = velocity

	// Update car position, size and rotation (ensure thread safety)
	fyne.Do(func() {
		w.updateCarTransform(vehicle, pixel)
	})
}

// removeVehicle removes a vehicle from the map
func (w *RoadMapWidget) removeVehicle(id string) {
	if vehicle, exists := w.vehicles[id]; exists {
		for _, part := range vehicle.carParts {
			part.Hide()
		}
		delete(w.vehicles, id)
	}
}

// createCarShape creates a realistic car using multiple rectangles
func (w *RoadMapWidget) createCarShape(fillColor color.Color, isEmergency bool) []*canvas.Rectangle {
	// Create car parts: body, windows, wheels
	carParts := make([]*canvas.Rectangle, 6) // 6 parts for a realistic car

	// Car body (main rectangle)
	carParts[0] = canvas.NewRectangle(fillColor)
	carParts[0].StrokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	carParts[0].StrokeWidth = 1

	// Front windshield
	carParts[1] = canvas.NewRectangle(color.RGBA{R: 135, G: 206, B: 235, A: 200}) // Sky blue, semi-transparent
	carParts[1].StrokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	carParts[1].StrokeWidth = 1

	// Rear windshield
	carParts[2] = canvas.NewRectangle(color.RGBA{R: 135, G: 206, B: 235, A: 200}) // Sky blue, semi-transparent
	carParts[2].StrokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	carParts[2].StrokeWidth = 1

	// Front wheel
	carParts[3] = canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255}) // Dark gray
	carParts[3].StrokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	carParts[3].StrokeWidth = 1

	// Rear wheel
	carParts[4] = canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255}) // Dark gray
	carParts[4].StrokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	carParts[4].StrokeWidth = 1

	// Emergency lights/roof (for ambulances) or roof detail
	if isEmergency {
		carParts[5] = canvas.NewRectangle(color.RGBA{R: 255, G: 255, B: 255, A: 255}) // White roof for ambulance
	} else {
		carParts[5] = canvas.NewRectangle(color.RGBA{R: 105, G: 105, B: 105, A: 255}) // Gray roof detail
	}
	carParts[5].StrokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	carParts[5].StrokeWidth = 1

	return carParts
}

// updateCarTransform updates the car position and direction based on velocity
func (w *RoadMapWidget) updateCarTransform(vehicle *VehicleMarker, pixel fyne.Position) {
	// Calculate direction from velocity
	direction := w.calculateDirection(vehicle.velocity)
	vehicle.direction = direction

	// Adjust size based on speed (velocity magnitude)
	speed := math.Sqrt(vehicle.velocity.Lat*vehicle.velocity.Lat + vehicle.velocity.Lng*vehicle.velocity.Lng)
	speedFactor := float32(math.Max(0.8, math.Min(1.5, speed*10))) // Scale factor between 0.8-1.5

	// Create directional car
	w.createDirectionalCar(vehicle, pixel, direction, speedFactor)
}

// calculateDirection maps velocity to one of four cardinal directions
func (w *RoadMapWidget) calculateDirection(velocity models.Velocity) string {
	// Calculate angle in radians from velocity components
	// Note: We need to account for the flipped Y coordinate in screen space
	angle := math.Atan2(-velocity.Lng, velocity.Lat) // Negative Lng because Y is flipped

	// Convert to degrees
	angleDegrees := angle * 180 / math.Pi
	if angleDegrees < 0 {
		angleDegrees += 360
	}

	// Map to cardinal directions with 45-degree ranges
	if angleDegrees >= 315 || angleDegrees < 45 {
		return "right" // East
	} else if angleDegrees >= 45 && angleDegrees < 135 {
		return "up" // North (up on screen)
	} else if angleDegrees >= 135 && angleDegrees < 225 {
		return "left" // West
	} else {
		return "down" // South (down on screen)
	}
}

// createDirectionalCar creates a car pointing in the specified direction
func (w *RoadMapWidget) createDirectionalCar(vehicle *VehicleMarker, pixel fyne.Position, direction string, speedFactor float32) {
	// Base car dimensions
	carLength := float32(20) * speedFactor
	carWidth := float32(12) * speedFactor

	// Hide all parts first
	for _, part := range vehicle.carParts {
		part.Hide()
	}

	switch direction {
	case "up":
		// Car pointing up (north)
		// Main body
		vehicle.carParts[0].Move(fyne.NewPos(pixel.X-carWidth/2, pixel.Y-carLength/2))
		vehicle.carParts[0].Resize(fyne.NewSize(carWidth, carLength))
		vehicle.carParts[0].Show()

		// Front windshield (top)
		vehicle.carParts[1].Move(fyne.NewPos(pixel.X-carWidth/2+2, pixel.Y-carLength/2+2))
		vehicle.carParts[1].Resize(fyne.NewSize(carWidth-4, carLength/3))
		vehicle.carParts[1].Show()

		// Rear windshield (bottom)
		vehicle.carParts[2].Move(fyne.NewPos(pixel.X-carWidth/2+2, pixel.Y+carLength/6))
		vehicle.carParts[2].Resize(fyne.NewSize(carWidth-4, carLength/3))
		vehicle.carParts[2].Show()

		// Front wheels (top)
		vehicle.carParts[3].Move(fyne.NewPos(pixel.X-carWidth/2-1, pixel.Y-carLength/3))
		vehicle.carParts[3].Resize(fyne.NewSize(3, carWidth/3))
		vehicle.carParts[3].Show()

		vehicle.carParts[4].Move(fyne.NewPos(pixel.X+carWidth/2-2, pixel.Y-carLength/3))
		vehicle.carParts[4].Resize(fyne.NewSize(3, carWidth/3))
		vehicle.carParts[4].Show()

		// Roof detail
		vehicle.carParts[5].Move(fyne.NewPos(pixel.X-carWidth/2+1, pixel.Y-carLength/4))
		vehicle.carParts[5].Resize(fyne.NewSize(carWidth-2, carLength/2))
		vehicle.carParts[5].Show()

	case "down":
		// Car pointing down (south)
		// Main body
		vehicle.carParts[0].Move(fyne.NewPos(pixel.X-carWidth/2, pixel.Y-carLength/2))
		vehicle.carParts[0].Resize(fyne.NewSize(carWidth, carLength))
		vehicle.carParts[0].Show()

		// Front windshield (bottom)
		vehicle.carParts[1].Move(fyne.NewPos(pixel.X-carWidth/2+2, pixel.Y+carLength/6))
		vehicle.carParts[1].Resize(fyne.NewSize(carWidth-4, carLength/3))
		vehicle.carParts[1].Show()

		// Rear windshield (top)
		vehicle.carParts[2].Move(fyne.NewPos(pixel.X-carWidth/2+2, pixel.Y-carLength/2+2))
		vehicle.carParts[2].Resize(fyne.NewSize(carWidth-4, carLength/3))
		vehicle.carParts[2].Show()

		// Front wheels (bottom)
		vehicle.carParts[3].Move(fyne.NewPos(pixel.X-carWidth/2-1, pixel.Y+carLength/6))
		vehicle.carParts[3].Resize(fyne.NewSize(3, carWidth/3))
		vehicle.carParts[3].Show()

		vehicle.carParts[4].Move(fyne.NewPos(pixel.X+carWidth/2-2, pixel.Y+carLength/6))
		vehicle.carParts[4].Resize(fyne.NewSize(3, carWidth/3))
		vehicle.carParts[4].Show()

		// Roof detail
		vehicle.carParts[5].Move(fyne.NewPos(pixel.X-carWidth/2+1, pixel.Y-carLength/4))
		vehicle.carParts[5].Resize(fyne.NewSize(carWidth-2, carLength/2))
		vehicle.carParts[5].Show()

	case "left":
		// Car pointing left (west)
		// Main body
		vehicle.carParts[0].Move(fyne.NewPos(pixel.X-carLength/2, pixel.Y-carWidth/2))
		vehicle.carParts[0].Resize(fyne.NewSize(carLength, carWidth))
		vehicle.carParts[0].Show()

		// Front windshield (left)
		vehicle.carParts[1].Move(fyne.NewPos(pixel.X-carLength/2+2, pixel.Y-carWidth/2+2))
		vehicle.carParts[1].Resize(fyne.NewSize(carLength/3, carWidth-4))
		vehicle.carParts[1].Show()

		// Rear windshield (right)
		vehicle.carParts[2].Move(fyne.NewPos(pixel.X+carLength/6, pixel.Y-carWidth/2+2))
		vehicle.carParts[2].Resize(fyne.NewSize(carLength/3, carWidth-4))
		vehicle.carParts[2].Show()

		// Front wheels (left)
		vehicle.carParts[3].Move(fyne.NewPos(pixel.X-carLength/3, pixel.Y-carWidth/2-1))
		vehicle.carParts[3].Resize(fyne.NewSize(carLength/3, 3))
		vehicle.carParts[3].Show()

		vehicle.carParts[4].Move(fyne.NewPos(pixel.X-carLength/3, pixel.Y+carWidth/2-2))
		vehicle.carParts[4].Resize(fyne.NewSize(carLength/3, 3))
		vehicle.carParts[4].Show()

		// Roof detail
		vehicle.carParts[5].Move(fyne.NewPos(pixel.X-carLength/4, pixel.Y-carWidth/2+1))
		vehicle.carParts[5].Resize(fyne.NewSize(carLength/2, carWidth-2))
		vehicle.carParts[5].Show()

	case "right":
		// Car pointing right (east)
		// Main body
		vehicle.carParts[0].Move(fyne.NewPos(pixel.X-carLength/2, pixel.Y-carWidth/2))
		vehicle.carParts[0].Resize(fyne.NewSize(carLength, carWidth))
		vehicle.carParts[0].Show()

		// Front windshield (right)
		vehicle.carParts[1].Move(fyne.NewPos(pixel.X+carLength/6, pixel.Y-carWidth/2+2))
		vehicle.carParts[1].Resize(fyne.NewSize(carLength/3, carWidth-4))
		vehicle.carParts[1].Show()

		// Rear windshield (left)
		vehicle.carParts[2].Move(fyne.NewPos(pixel.X-carLength/2+2, pixel.Y-carWidth/2+2))
		vehicle.carParts[2].Resize(fyne.NewSize(carLength/3, carWidth-4))
		vehicle.carParts[2].Show()

		// Front wheels (right)
		vehicle.carParts[3].Move(fyne.NewPos(pixel.X+carLength/6, pixel.Y-carWidth/2-1))
		vehicle.carParts[3].Resize(fyne.NewSize(carLength/3, 3))
		vehicle.carParts[3].Show()

		vehicle.carParts[4].Move(fyne.NewPos(pixel.X+carLength/6, pixel.Y+carWidth/2-2))
		vehicle.carParts[4].Resize(fyne.NewSize(carLength/3, 3))
		vehicle.carParts[4].Show()

		// Roof detail
		vehicle.carParts[5].Move(fyne.NewPos(pixel.X-carLength/4, pixel.Y-carWidth/2+1))
		vehicle.carParts[5].Resize(fyne.NewSize(carLength/2, carWidth-2))
		vehicle.carParts[5].Show()
	}
}

// getVehicleColor returns a vibrant color for each vehicle
func (w *RoadMapWidget) getVehicleColor(id string, isEmergency bool) color.Color {
	// Emergency vehicles get distinctive white color (ambulance)
	if isEmergency {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // White for ambulance
	}

	// Predefined vibrant colors for regular vehicles
	vibrantColors := []color.Color{
		color.RGBA{R: 30, G: 144, B: 255, A: 255},  // Dodger Blue
		color.RGBA{R: 50, G: 205, B: 50, A: 255},   // Lime Green
		color.RGBA{R: 255, G: 215, B: 0, A: 255},   // Gold
		color.RGBA{R: 138, G: 43, B: 226, A: 255},  // Blue Violet
		color.RGBA{R: 0, G: 255, B: 255, A: 255},   // Cyan
		color.RGBA{R: 255, G: 105, B: 180, A: 255}, // Hot Pink
		color.RGBA{R: 124, G: 252, B: 0, A: 255},   // Lawn Green
		color.RGBA{R: 255, G: 20, B: 147, A: 255},  // Deep Pink
	}

	// Generate a hash from the vehicle ID
	hash := 0
	for _, char := range id {
		hash = int(char) + ((hash << 5) - hash)
	}

	// Use hash to select a vibrant color
	colorIndex := hash % len(vibrantColors)
	if colorIndex < 0 {
		colorIndex = -colorIndex
	}

	return vibrantColors[colorIndex]
}
