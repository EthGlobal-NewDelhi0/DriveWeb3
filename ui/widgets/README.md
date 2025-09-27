# Road Map Widget

This package contains the road map visualization for the PeerDrive mesh network.

## Map Specifications

### Grid Layout
- **2x2 Grid**: Cartesian coordinate system from (0,0) to (2,2)
- **3 Vertical Roads**: 
  - Road 1: X = 0.0 to 0.4
  - Road 2: X = 0.8 to 1.2  
  - Road 3: X = 1.6 to 2.0
- **2 Horizontal Roads**: 
  - Road 1: Y = 0.0 to 0.4
  - Road 2: Y = 1.6 to 2.0

### Road Design
- **Road Width**: 0.4 units each
- **Lane Configuration**: 2 lanes per road (bidirectional)
- **Lane Dividers**: White dotted center lines
- **Road Color**: Dark gray (#3C3C3C)
- **Background**: Forest green (#228B22)

### Coordinate System
- **World Coordinates**: (0,0) to (2,2) floating point
- **Origin**: Bottom-left corner (0,0)
- **X-Axis**: Left to right
- **Y-Axis**: Bottom to top (standard Cartesian)

### Visual Features
- **Lane Markings**: White dotted center dividers on all roads
- **Intersections**: Roads cross naturally at specified coordinates
- **Clean Design**: No grid lines, clean green background
- **Responsive**: Maintains square aspect ratio when resized

## Usage

```go
// Create a new road map widget
roadMap := widgets.NewRoadMapWidget()

// Get world coordinate bounds
minX, minY, maxX, maxY := roadMap.GetWorldBounds()

// Convert between coordinate systems
pixelPos := roadMap.worldToPixel(2.0, 1.5)
worldX, worldY := roadMap.pixelToWorld(pixelPos.X, pixelPos.Y)

// Future: Add vehicle markers
roadMap.AddVehicle("vehicle-123", 1.2, 3.4)
```

## Road Network Layout

```
2.0 ░░░░░░│░░░░░░│░░░░░░
1.6 ──────┼──────┼──────  ← Horizontal Road 2
    ░░░░░░│░░░░░░│░░░░░░
    ░░░░░░│░░░░░░│░░░░░░
0.4 ──────┼──────┼──────  ← Horizontal Road 1  
0.0 ░░░░░░│░░░░░░│░░░░░░
    0.0   0.8   1.6   2.0
    │     │     │
    Road1 Road2 Road3
   (V)   (V)   (V)

░ = Green terrain
─ = Horizontal roads
│ = Vertical roads
┼ = Intersections
```

## Future Enhancements

1. **Vehicle Markers**: Add moving dots representing mesh network peers
2. **Traffic Flow**: Color-coded arrows showing traffic density
3. **Connection Lines**: Visual representation of P2P connections
4. **Interactive Features**: Click to show vehicle details
5. **Real-time Updates**: Animate vehicle movement based on sensor data

## Integration

The road map is integrated into the main UI's right panel and provides the foundation for visualizing the vehicle mesh network in real-time.

