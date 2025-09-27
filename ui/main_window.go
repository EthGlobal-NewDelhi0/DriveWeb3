package ui

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"peerdrive/app/pkg/commands"
	"peerdrive/app/pkg/demo"
	"peerdrive/app/pkg/hardware"
	"peerdrive/app/pkg/network"
	"peerdrive/app/ui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// MainWindow represents the main UI window
type MainWindow struct {
	app           fyne.App
	window        fyne.Window
	handlers      *commands.CommandHandlers
	ctx           context.Context
	outputText    *widget.RichText
	peerAddrLabel *widget.Label
	roadMap       *widgets.RoadMapWidget
	netManager    *network.MeshNetworkManager // Add reference to network manager
	demoScenario  *demo.AmbulanceScenario     // Ambulance demo scenario
}

// NewMainWindow creates a new main window
func NewMainWindow(ctx context.Context, netManager *network.MeshNetworkManager, cryptoManager *hardware.HardwareManager, rpcURL, privateKey, tokenAddress string) *MainWindow {
	fyneApp := app.New()
	fyneApp.SetIcon(nil) // You can set an icon later

	window := fyneApp.NewWindow("PeerDrive - Vehicle Mesh Network")
	window.SetTitle("PeerDrive - Vehicle Mesh Network")

	handlers := commands.NewCommandHandlers(netManager, cryptoManager, rpcURL, privateKey, tokenAddress)

	mw := &MainWindow{
		app:          fyneApp,
		window:       window,
		handlers:     handlers,
		ctx:          ctx,
		netManager:   netManager,
		demoScenario: demo.NewAmbulanceScenario(),
	}

	mw.setupUI()
	return mw
}

// setupUI initializes the user interface
func (mw *MainWindow) setupUI() {
	// Window configuration
	mw.window.Resize(fyne.NewSize(1200, 800))
	mw.window.SetContent(mw.createContent())

	// Start map updates
	go mw.startMapUpdates()
}

// createContent creates the main content layout
func (mw *MainWindow) createContent() fyne.CanvasObject {
	// Create the left and right panels
	leftPanel := mw.createLeftPanel()
	rightPanel := mw.createRightPanel()

	// Create split container
	splitContainer := container.NewHSplit(leftPanel, rightPanel)
	splitContainer.SetOffset(0.6) // 60% left, 40% right

	return splitContainer
}

// createLeftPanel creates the left control panel
func (mw *MainWindow) createLeftPanel() fyne.CanvasObject {
	// Peer address display at top
	peerAddr := mw.handlers.GetPeerAddress()
	mw.peerAddrLabel = widget.NewLabel(fmt.Sprintf("Peer Address: %s", peerAddr))
	mw.peerAddrLabel.Wrapping = fyne.TextWrapWord

	// Command buttons section
	buttonsContainer := mw.createButtonsSection()

	// Output display section
	outputContainer := mw.createOutputSection()

	// Combine everything in a vertical box
	leftContent := container.NewVBox(
		widget.NewCard("Peer Information", "", mw.peerAddrLabel),
		widget.NewCard("Commands", "", buttonsContainer),
		widget.NewCard("Output", "", outputContainer),
	)

	// Make it scrollable
	scroll := container.NewScroll(leftContent)
	scroll.SetMinSize(fyne.NewSize(400, 600))

	return scroll
}

// createButtonsSection creates the buttons for all commands
func (mw *MainWindow) createButtonsSection() fyne.CanvasObject {
	// Network commands
	addPeerBtn := widget.NewButton("Add Peer", mw.showAddPeerDialog)
	removePeerBtn := widget.NewButton("Remove Peer", mw.showRemovePeerDialog)
	showPeersBtn := widget.NewButton("Show Peers", mw.handleShowPeers)

	// Information commands
	showRewardsBtn := widget.NewButton("Show Accrued Rewards", mw.handleShowRewards)
	showNonceBtn := widget.NewButton("Show Nonce", mw.handleShowNonce)
	showLedgerBtn := widget.NewButton("Show Ledger", mw.handleShowLedger)
	showEthAddrBtn := widget.NewButton("Show ETH Address", mw.handleShowEthAddress)

	// Blockchain commands
	approveTokenBtn := widget.NewButton("Approve Token", mw.showApproveTokenDialog)
	redeemBtn := widget.NewButton("Redeem Rewards", mw.handleRedeem)

	// Utility commands
	clearBtn := widget.NewButton("Clear Output", mw.handleClear)
	refreshBtn := widget.NewButton("Refresh Peer Info", mw.handleRefresh)
	showVehiclesBtn := widget.NewButton("Show Vehicle Count", mw.handleShowVehicles)

	// Demo commands
	demoNoV2VBtn := widget.NewButton("🚨 Demo: No V2V", mw.handleDemoNoV2V)
	demoWithV2VBtn := widget.NewButton("📡 Demo: With V2V", mw.handleDemoWithV2V)
	stopDemoBtn := widget.NewButton("⏹️ Stop Demo", mw.handleStopDemo)

	// Organize buttons in a grid
	buttonsGrid := container.NewGridWithColumns(2,
		addPeerBtn, removePeerBtn,
		showPeersBtn, showRewardsBtn,
		showNonceBtn, showLedgerBtn,
		showEthAddrBtn, approveTokenBtn,
		redeemBtn, clearBtn,
		refreshBtn, showVehiclesBtn,
		demoNoV2VBtn, demoWithV2VBtn,
		stopDemoBtn, widget.NewLabel(""), // Empty cell for alignment
	)

	return buttonsGrid
}

// createOutputSection creates the output display area
func (mw *MainWindow) createOutputSection() fyne.CanvasObject {
	mw.outputText = widget.NewRichText()
	mw.outputText.Wrapping = fyne.TextWrapWord
	mw.appendOutput("🚀 PeerDrive UI started. Ready for commands!")

	// Create scrollable container for output
	outputScroll := container.NewScroll(mw.outputText)
	outputScroll.SetMinSize(fyne.NewSize(380, 300))

	return outputScroll
}

// createRightPanel creates the right panel with road map
func (mw *MainWindow) createRightPanel() fyne.CanvasObject {
	// Create the road map widget
	mw.roadMap = widgets.NewRoadMapWidget()

	// Create info label
	mapInfo := widget.NewLabel("2x2 Road Grid | 3 Vertical + 2 Horizontal Roads | 0.4 units wide each")
	mapInfo.Alignment = fyne.TextAlignCenter
	mapInfo.Wrapping = fyne.TextWrapWord

	// Combine map and info in a vertical container
	mapContainer := container.NewVBox(
		mapInfo,
		mw.roadMap,
	)

	mapCard := widget.NewCard("Network Map", "Real-time vehicle mesh network", mapContainer)

	return container.NewScroll(mapCard)
}

// Button handlers

func (mw *MainWindow) showAddPeerDialog() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter peer multiaddress (e.g., /ip4/192.168.1.100/tcp/4001/p2p/...)")
	entry.MultiLine = false

	dialog.ShowForm("Add Peer", "Connect", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Peer Address", entry),
	}, func(confirmed bool) {
		if confirmed && entry.Text != "" {
			result := mw.handlers.AddPeer(entry.Text)
			mw.displayCommandResult("Add Peer", result)
		}
	}, mw.window)
}

func (mw *MainWindow) showRemovePeerDialog() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter peer ID to disconnect")

	dialog.ShowForm("Remove Peer", "Disconnect", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Peer ID", entry),
	}, func(confirmed bool) {
		if confirmed && entry.Text != "" {
			result := mw.handlers.RemovePeer(entry.Text)
			mw.displayCommandResult("Remove Peer", result)
		}
	}, mw.window)
}

func (mw *MainWindow) handleShowPeers() {
	result := mw.handlers.ShowPeers()
	mw.displayCommandResult("Show Peers", result)
}

func (mw *MainWindow) handleShowRewards() {
	result := mw.handlers.ShowAccruedRewards()
	mw.displayCommandResult("Show Rewards", result)
}

func (mw *MainWindow) handleShowNonce() {
	result := mw.handlers.ShowNonce()
	mw.displayCommandResult("Show Nonce", result)
}

func (mw *MainWindow) handleShowLedger() {
	result := mw.handlers.ShowLedger()
	mw.displayCommandResult("Show Ledger", result)
}

func (mw *MainWindow) handleShowEthAddress() {
	result := mw.handlers.ShowEthAddress()
	mw.displayCommandResult("Show ETH Address", result)
}

func (mw *MainWindow) showApproveTokenDialog() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter spender address (0x...)")

	dialog.ShowForm("Approve Token", "Approve", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Spender Address", entry),
	}, func(confirmed bool) {
		if confirmed && entry.Text != "" {
			result := mw.handlers.ApproveToken(mw.ctx, entry.Text)
			mw.displayCommandResult("Approve Token", result)
		}
	}, mw.window)
}

func (mw *MainWindow) handleRedeem() {
	result := mw.handlers.RedeemRewards(mw.ctx)
	mw.displayCommandResult("Redeem Rewards", result)
}

func (mw *MainWindow) handleClear() {
	mw.outputText.ParseMarkdown("")
	mw.appendOutput("🧹 Output cleared!")
}

func (mw *MainWindow) handleRefresh() {
	peerAddr := mw.handlers.GetPeerAddress()
	mw.peerAddrLabel.SetText(fmt.Sprintf("Peer Address: %s", peerAddr))

	// Also show map coordinates info
	minX, minY, maxX, maxY := mw.roadMap.GetWorldBounds()
	mapInfo := fmt.Sprintf("🗺️ Map bounds: (%.1f,%.1f) to (%.1f,%.1f) | Vertical roads: 0-0.4, 0.8-1.2, 1.6-2.0", minX, minY, maxX, maxY)

	mw.appendOutput("🔄 Peer information refreshed!")
	mw.appendOutput(mapInfo)
}

func (mw *MainWindow) handleShowVehicles() {
	ledger := mw.netManager.GetLedger()
	vehicleCount := len(ledger.Ledger)

	if vehicleCount == 0 {
		mw.appendOutput("🚗 No vehicles currently on the map")
		return
	}

	mw.appendOutput(fmt.Sprintf("🚗 %d vehicle(s) on the map:", vehicleCount))

	for peerID, entry := range ledger.Ledger {
		pos := entry.VehicleState.Position
		shortID := peerID
		if len(peerID) > 12 {
			shortID = peerID[:12] + "..."
		}
		mw.appendOutput(fmt.Sprintf("  • %s at (%.2f, %.2f)", shortID, pos.Lat, pos.Lng))
	}
}

// Utility methods

func (mw *MainWindow) displayCommandResult(command string, result commands.CommandResult) {
	timestamp := time.Now().Format("15:04:05")
	status := "✅"
	if !result.Success {
		status = "❌"
	}

	output := fmt.Sprintf("**[%s] %s %s:**\n%s\n", timestamp, status, command, result.Message)
	if result.Error != nil {
		output += fmt.Sprintf("*Error: %v*\n", result.Error)
	}
	output += "\n---\n"

	mw.appendOutput(output)
}

func (mw *MainWindow) appendOutput(text string) {
	// Get current content and append new text
	currentContent := mw.outputText.String()
	mw.outputText.ParseMarkdown(currentContent + text + "\n")
}

// Run starts the UI application
func (mw *MainWindow) Run() {
	log.Println("🖥️  Starting PeerDrive GUI...")
	mw.window.ShowAndRun()
}

// SetCloseIntercept sets a function to call when the window is closing
func (mw *MainWindow) SetCloseIntercept(onClose func()) {
	mw.window.SetCloseIntercept(onClose)
}

// startMapUpdates periodically updates the map with vehicle positions from the ledger
func (mw *MainWindow) startMapUpdates() {
	ticker := time.NewTicker(time.Second / 5) // Update 5 times per second
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if demo scenario is running
			if mw.demoScenario.IsRunning() {
				// Use demo scenario data
				ledger := mw.demoScenario.GetCurrentState()
				mw.roadMap.UpdateVehicles(ledger)
			} else {
				// Get current ledger state from network
				ledger := mw.netManager.GetLedger()
				mw.roadMap.UpdateVehicles(ledger)
			}

		case <-mw.ctx.Done():
			return
		}
	}
}

// Demo handler methods

func (mw *MainWindow) handleDemoNoV2V() {
	if mw.demoScenario.IsRunning() {
		mw.appendOutput("❗ Demo already running. Stop current demo first.")
		return
	}

	err := mw.demoScenario.StartScenario(demo.NoV2V)
	if err != nil {
		mw.appendOutput(fmt.Sprintf("❌ Failed to start demo: %v", err))
		return
	}

	mw.appendOutput("🚨 Starting Emergency Demo WITHOUT V2V Communication")
	mw.appendOutput("➡️ Normal car ahead, ambulance behind - no communication")
	mw.appendOutput("⚠️ Ambulance will be stuck behind normal car...")

	// Start monitoring demo progress
	go mw.monitorDemoProgress()
}

func (mw *MainWindow) handleDemoWithV2V() {
	if mw.demoScenario.IsRunning() {
		mw.appendOutput("❗ Demo already running. Stop current demo first.")
		return
	}

	err := mw.demoScenario.StartScenario(demo.WithV2V)
	if err != nil {
		mw.appendOutput(fmt.Sprintf("❌ Failed to start demo: %v", err))
		return
	}

	mw.appendOutput("📡 Starting Emergency Demo WITH V2V Communication")
	mw.appendOutput("➡️ Normal car ahead, ambulance behind - V2V enabled")
	mw.appendOutput("🚗 Car will receive ambulance alert and move aside...")

	// Start monitoring demo progress
	go mw.monitorDemoProgress()
}

func (mw *MainWindow) handleStopDemo() {
	if !mw.demoScenario.IsRunning() {
		mw.appendOutput("❗ No demo currently running.")
		return
	}

	result := mw.demoScenario.StopScenario()
	if result != nil {
		mw.displayDemoResult(result)
	} else {
		mw.appendOutput("⏹️ Demo stopped.")
	}
}

// monitorDemoProgress monitors the demo scenario and provides updates
func (mw *MainWindow) monitorDemoProgress() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for mw.demoScenario.IsRunning() {
		select {
		case <-ticker.C:
			_ = mw.demoScenario.GetScenarioStatus()
			// Update status in UI (could be a status label if added)

		case <-time.After(30 * time.Second): // Timeout after 30 seconds
			mw.appendOutput("⏰ Demo timeout - stopping scenario")
			result := mw.demoScenario.StopScenario()
			if result != nil {
				mw.displayDemoResult(result)
			}
			return
		}
	}

	// Demo completed naturally
	result := mw.demoScenario.StopScenario()
	if result != nil {
		mw.displayDemoResult(result)
	}
}

// displayDemoResult shows the results of a completed demo
func (mw *MainWindow) displayDemoResult(result *demo.ScenarioResult) {
	mw.appendOutput("\n" + strings.Repeat("=", 50))
	mw.appendOutput("📊 DEMO RESULTS")
	mw.appendOutput(strings.Repeat("=", 50))

	scenarioName := "Without V2V Communication"
	if result.ScenarioType == demo.WithV2V {
		scenarioName = "With V2V Communication"
	}

	mw.appendOutput(fmt.Sprintf("🎯 Scenario: %s", scenarioName))
	mw.appendOutput(fmt.Sprintf("⏱️ Ambulance Journey Time: %.1f seconds", result.AmbulanceTime.Seconds()))
	mw.appendOutput(fmt.Sprintf("🚑 Average Ambulance Speed: %.1f m/s", result.AverageAmbSpeed))

	if result.V2VAlertSent {
		mw.appendOutput("📡 V2V Alert: ✅ Sent")
	} else {
		mw.appendOutput("📡 V2V Alert: ❌ Not sent")
	}

	if result.CarYieldedSpace {
		mw.appendOutput("🚗 Car Response: ✅ Yielded space")
	} else {
		mw.appendOutput("🚗 Car Response: ❌ No response")
	}

	mw.appendOutput(fmt.Sprintf("🏥 Patient Outcome: %s", result.PatientOutcome))
	mw.appendOutput(strings.Repeat("=", 50))

	// Add comparison hint
	if result.ScenarioType == demo.NoV2V {
		mw.appendOutput("💡 Try running 'Demo: With V2V' to see the difference!")
	} else {
		mw.appendOutput("💡 Compare with 'Demo: No V2V' to see the improvement!")
	}
}

// Close gracefully shuts down the UI
func (mw *MainWindow) Close() {
	// Stop any running demo
	if mw.demoScenario.IsRunning() {
		mw.demoScenario.StopScenario()
	}
	mw.window.Close()
}
