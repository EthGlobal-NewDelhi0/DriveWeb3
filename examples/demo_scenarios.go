package main

import (
	"fmt"
	"log"
	"time"

	"peerdrive/app/pkg/models"
	"peerdrive/app/pkg/strategy"
)

func main() {
	fmt.Println("🚗 PeerDrive Strategy Scenarios Demo")
	fmt.Println("====================================")

	// Initialize network state
	networkState := &models.NetworkState{
		Peers:                  make(map[string]models.VehicleState),
		EmergencyAlerts:        make([]string, 0),
		TrafficRecommendations: "",
	}

	// Create strategy engine
	engine := strategy.NewEngine(networkState)

	// Demo 1: 5-Vehicle Stopping without V2V communication
	fmt.Println("\n🚦 Demo 1: 5-Vehicle Stopping (No Communication)")
	fmt.Println("-----------------------------------------------")
	runScenario(engine, strategy.ScenarioFiveVehicleStopping, strategy.NoComm, "five_vehicle_no_comm")

	// Wait a moment between scenarios
	time.Sleep(2 * time.Second)

	// Demo 2: 5-Vehicle Stopping with V2V communication
	fmt.Println("\n🚦 Demo 2: 5-Vehicle Stopping (V2V Communication)")
	fmt.Println("--------------------------------------------------")
	runScenario(engine, strategy.ScenarioFiveVehicleStopping, strategy.V2VComm, "five_vehicle_v2v")

	// Wait a moment between scenarios
	time.Sleep(2 * time.Second)

	// Demo 3: Emergency Vehicle Priority
	fmt.Println("\n🚨 Demo 3: Emergency Vehicle Priority")
	fmt.Println("-------------------------------------")
	runScenario(engine, strategy.ScenarioEmergencyVehicle, strategy.V2VComm, "emergency_vehicle")

	fmt.Println("\n✅ All demos completed! Check the generated files for detailed metrics.")
}

func runScenario(engine *strategy.Engine, scenarioType strategy.ScenarioType, commMode strategy.CommunicationMode, filePrefix string) {
	// Start the scenario
	err := engine.StartScenario(scenarioType, commMode)
	if err != nil {
		log.Printf("❌ Failed to start scenario: %v", err)
		return
	}

	fmt.Printf("▶️  Scenario started (%s mode)\n", getCommunicationModeString(commMode))

	// Monitor progress
	startTime := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !engine.IsRunning() {
				// Scenario completed
				duration := time.Since(startTime)
				fmt.Printf("🏁 Scenario completed in %.1f seconds\n", duration.Seconds())

				// Get final metrics
				metrics := engine.StopScenario()
				printMetricsSummary(metrics)

				// Save metrics to files
				jsonFile := strategy.GenerateTimestampedFilename(filePrefix, "json")
				csvFile := strategy.GenerateTimestampedFilename(filePrefix, "csv")

				if err := strategy.SaveMetricsToJSON(metrics, scenarioType, commMode, duration, jsonFile); err != nil {
					log.Printf("⚠️  Failed to save JSON: %v", err)
				} else {
					fmt.Printf("💾 Saved detailed metrics to: %s\n", jsonFile)
				}

				if err := strategy.SaveMetricsToCSV(metrics, scenarioType, commMode, csvFile); err != nil {
					log.Printf("⚠️  Failed to save CSV: %v", err)
				} else {
					fmt.Printf("📊 Saved CSV data to: %s\n", csvFile)
				}

				return
			}

			// Display current status
			_, vehicles, _ := engine.GetCurrentScenarioState()
			printVehicleStatus(vehicles)

		case <-time.After(30 * time.Second):
			// Timeout after 30 seconds
			fmt.Println("⏰ Scenario timeout - stopping...")
			engine.StopScenario()
			return
		}
	}
}

func printVehicleStatus(vehicles map[string]*strategy.VehicleScenarioState) {
	if len(vehicles) == 0 {
		return
	}

	fmt.Print("📊 Status: ")
	for id, vehicle := range vehicles {
		alertIcon := ""
		if vehicle.ReceivedV2VAlert {
			alertIcon = "📡"
		}
		emergencyIcon := ""
		if vehicle.IsEmergency {
			emergencyIcon = "🚨"
		}

		fmt.Printf("%s%s%s(%.0fm,%.1fm/s) ", id, alertIcon, emergencyIcon, vehicle.Position, vehicle.Velocity)
	}
	fmt.Println()
}

func printMetricsSummary(metrics []strategy.ScenarioMetrics) {
	if len(metrics) == 0 {
		return
	}

	collisions := 0
	avgTTC := 0.0
	validTTC := 0
	avgDecel := 0.0
	avgV2VResp := 0.0
	validV2V := 0

	for _, metric := range metrics {
		if metric.CollisionOccurred {
			collisions++
		}
		if metric.TimeToCollision < 1000 {
			avgTTC += metric.TimeToCollision
			validTTC++
		}
		avgDecel += metric.PeakDeceleration
		if metric.V2VResponseTime > 0 {
			avgV2VResp += metric.V2VResponseTime
			validV2V++
		}
	}

	if validTTC > 0 {
		avgTTC /= float64(validTTC)
	}
	avgDecel /= float64(len(metrics))
	if validV2V > 0 {
		avgV2VResp /= float64(validV2V)
	}

	fmt.Printf("📈 Results: %d vehicles, %d collisions, %.1fs avg TTC, %.1f m/s² avg decel",
		len(metrics), collisions, avgTTC, avgDecel)
	if validV2V > 0 {
		fmt.Printf(", %.0fms avg V2V response", avgV2VResp*1000)
	}
	fmt.Println()
}

func getCommunicationModeString(mode strategy.CommunicationMode) string {
	switch mode {
	case strategy.NoComm:
		return "No Communication"
	case strategy.V2VComm:
		return "V2V Communication"
	default:
		return "Unknown"
	}
}
