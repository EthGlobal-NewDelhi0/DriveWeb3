package main

import (
	"context"
	"log"
	"os"
	"time"

	"peerdrive/app/pkg/hardware"
	"peerdrive/app/pkg/models"
	"peerdrive/app/pkg/network"
	"peerdrive/app/pkg/sensor"
	"peerdrive/app/ui"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file before reading environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found or failed to load, proceeding with existing environment variables")
	}

	rpcURL := os.Getenv("RPC_URL")
	privateKey := os.Getenv("PRIVATE_KEY")
	tokenAddress := os.Getenv("TOKEN_ADDRESS")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize crypto hardware manager
	cryptoManager, err := hardware.NewHardwareManager(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize crypto manager: %v", err)
	}
	defer cryptoManager.Close()

	// Initialize Network Manager
	netManager, err := network.NewMeshNetworkManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create network manager: %v", err)
	}
	netManager.StartMessageHandler(ctx)

	// Initialize Mock Sensor
	sensorManager := sensor.NewMockManager(netManager.GetPeerId())
	vehicleStateChan, err := sensorManager.StartTracking()
	if err != nil {
		log.Fatalf("Failed to start sensor manager: %v", err)
	}
	defer sensorManager.StopTracking()

	// Create and run the GUI first
	mainWindow := ui.NewMainWindow(ctx, netManager, cryptoManager, rpcURL, privateKey, tokenAddress)

	// Set up cleanup when window closes
	mainWindow.SetCloseIntercept(func() {
		log.Println("Window closing, shutting down background processes...")
		cancel() // Cancel context to stop all goroutines
		mainWindow.Close()
	})

	// Start background sensor data processing and network operations
	log.Println("🚀 Starting background processes...")
	go handleNetworkAndSensorLoop(ctx, netManager, vehicleStateChan)

	// Give background processes a moment to start
	time.Sleep(100 * time.Millisecond)
	log.Println("✅ Background processes started")

	// Run the GUI (blocking)
	mainWindow.Run()

	log.Println("Shutting down PeerDrive GUI...")
}

// handleNetworkAndSensorLoop combines sensor data processing and network operations
// This mirrors the main loop structure from cmd/peerdrive/main.go
func handleNetworkAndSensorLoop(ctx context.Context, netManager *network.MeshNetworkManager, vehicleStateChan <-chan models.VehicleState) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("💥 PANIC in network/sensor loop: %v", r)
		}
		log.Println("✅ Network and sensor loop stopped")
	}()

	log.Println("🔄 Starting network and sensor loop...")

	// Use same timing as the working CLI version: 2Hz instead of 5Hz
	gossipTicker := time.NewTicker(time.Second / 4) // 2 times per second (like CLI)
	defer gossipTicker.Stop()

	log.Println("🚀 Network and sensor loop started. Waiting for data...")

	for {
		select {
		case stateUpdate := <-vehicleStateChan:
			// A new state was received from our mock sensor
			netManager.UpdateLocalVehicleState(stateUpdate)

		case <-gossipTicker.C:
			// Periodic network operations
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("💥 PANIC in gossip operation: %v", r)
					}
				}()
				netManager.Gossip()
				netManager.StoreLedger() // Re-enable this now that we have proper error handling
			}()

		case <-ctx.Done():
			log.Println("🛑 Network and sensor loop shutting down...")
			return
		}
	}
}
