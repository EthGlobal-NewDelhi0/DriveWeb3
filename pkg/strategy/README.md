# PeerDrive Strategy Engine

This package implements V2V communication scenarios to demonstrate the safety and efficiency advantages of vehicular mesh networking.

## Overview

The strategy engine provides two main simulation scenarios:

1. **5-Vehicle Stopping Scenario**: Compares reaction to emergency braking with and without V2V communication
2. **Emergency Vehicle Priority**: Demonstrates cooperative lane-changing for emergency vehicle passage

## Scenarios

### 1. Five-Vehicle Stopping Scenario

This scenario simulates a line of 5 vehicles where the lead vehicle performs emergency braking.

**Setup:**
- 5 vehicles in a single lane
- Initial speed: 54 km/h (15 m/s)
- Uniform headway: 30 meters
- Safe following distance: 20 meters
- Human reaction time: 1.5 seconds
- Maximum braking: 8 m/s²

**Modes:**
- **No Communication (Baseline)**: Each vehicle reacts only when it visually observes the vehicle ahead braking
- **V2V Communication**: Lead vehicle broadcasts instant emergency alert with 50ms network latency

**Metrics Collected:**
- Time-to-Collision (TTC)
- Peak deceleration experienced
- Spacing error from safe headway
- Collision occurrence
- V2V response time

### 2. Emergency Vehicle Priority Scenario

This scenario simulates an ambulance approaching two vehicles from behind.

**Setup:**
- 3 vehicles: 2 regular cars + 1 ambulance
- Initial spacing: 40 meters
- Ambulance starts behind both vehicles
- V2V emergency signal broadcast with 30ms latency

**Behavior:**
- Ambulance increases speed and broadcasts emergency signal
- Regular vehicles receive signal and change to left lane
- System measures time for ambulance to pass through

**Metrics Collected:**
- V2V response time for lane changes
- Emergency vehicle passage time
- Vehicle positions and lane changes over time

## Usage

### GUI Interface

Use the PeerDrive GUI application:

1. Start the GUI: `go run cmd/peerdrive-gui/main.go`
2. Click scenario buttons:
   - "5-Vehicle Stop (No Comm)" - Run baseline scenario
   - "5-Vehicle Stop (V2V)" - Run V2V-enabled scenario
   - "Emergency Vehicle" - Run emergency priority scenario
   - "Stop Scenario" - Stop current simulation
   - "Show Metrics" - Display current metrics

### Command Line Demo

Run the example script:

```bash
cd examples
go run demo_scenarios.go
```

This runs all scenarios automatically and saves results to timestamped files.

### Programmatic Usage

```go
import "peerdrive/app/pkg/strategy"

// Create network state and engine
networkState := &models.NetworkState{
    Peers: make(map[string]models.VehicleState),
    EmergencyAlerts: make([]string, 0),
    TrafficRecommendations: "",
}
engine := strategy.NewEngine(networkState)

// Run 5-vehicle scenario with V2V communication
err := engine.StartScenario(strategy.ScenarioFiveVehicleStopping, strategy.V2VComm)
if err != nil {
    log.Fatal(err)
}

// Monitor until completion
for engine.IsRunning() {
    time.Sleep(1 * time.Second)
    _, vehicles, metrics := engine.GetCurrentScenarioState()
    // Process current state...
}

// Get final metrics
finalMetrics := engine.StopScenario()
```

## Metrics and Analysis

### Key Performance Indicators

1. **Safety Metrics:**
   - Collision count (0 = best)
   - Minimum Time-to-Collision (higher = safer)
   - Peak deceleration (lower = more comfortable)

2. **Efficiency Metrics:**
   - V2V response time (lower = better)
   - Emergency passage time (lower = better)
   - Spacing error from optimal headway

3. **Safety Score:**
   - Composite score 0-100 based on multiple factors
   - Penalizes collisions, low TTC, harsh braking
   - Rewards fast V2V response

### Expected Results

**5-Vehicle Stopping:**
- **No Comm**: Higher collision risk, lower TTC, chain-reaction braking
- **V2V Comm**: No collisions, higher TTC, coordinated braking response

**Emergency Vehicle:**
- Demonstrates cooperative behavior enabling faster emergency response
- Measures effectiveness of V2V emergency signal propagation

### Data Export

Metrics are automatically saved in multiple formats:

- **JSON**: Complete scenario data with metadata
- **CSV**: Tabular data for statistical analysis
- **Console**: Real-time status and summary tables

## Implementation Details

### Physics Model

- Realistic vehicle dynamics with acceleration/deceleration limits
- Kinematic updates at 50Hz for smooth simulation
- Proper reaction time modeling for human drivers

### Communication Model

- Realistic network latency (30-50ms)
- Message propagation delays
- V2V alert reception and processing

### Scenario Control

- Deterministic scenarios for reproducible results
- Configurable vehicle parameters
- Automatic scenario termination conditions

## Files

- `engine.go`: Main strategy engine and scenario management
- `scenarios.go`: Specific scenario implementations  
- `analysis.go`: Metrics collection and export utilities
- `README.md`: This documentation

## Future Enhancements

1. **Additional Scenarios:**
   - Traffic wave dissipation
   - Intersection coordination
   - Merge assist

2. **Enhanced Physics:**
   - Vehicle-specific characteristics
   - Weather/road conditions
   - More sophisticated driver models

3. **Network Effects:**
   - Packet loss simulation
   - Network congestion modeling
   - Multi-hop message relay
