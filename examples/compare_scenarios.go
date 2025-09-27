package main

import (
	"fmt"
	"log"
	"os"

	"peerdrive/app/pkg/strategy"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run compare_scenarios.go <report1.json> <report2.json>")
		fmt.Println("")
		fmt.Println("Example:")
		fmt.Println("  go run compare_scenarios.go five_vehicle_no_comm_20240127_143022.json five_vehicle_v2v_20240127_143045.json")
		os.Exit(1)
	}

	report1File := os.Args[1]
	report2File := os.Args[2]

	// Load the first report
	report1, err := strategy.LoadMetricsFromJSON(report1File)
	if err != nil {
		log.Fatalf("❌ Failed to load %s: %v", report1File, err)
	}

	// Load the second report
	report2, err := strategy.LoadMetricsFromJSON(report2File)
	if err != nil {
		log.Fatalf("❌ Failed to load %s: %v", report2File, err)
	}

	// Generate comparison
	comparison := strategy.CompareScenarios(report1, report2)

	// Print to console
	fmt.Println("📊 PeerDrive Scenario Comparison")
	fmt.Println("================================")
	fmt.Println()
	fmt.Println(comparison)

	// Save to file
	outputFile := "scenario_comparison.md"
	err = os.WriteFile(outputFile, []byte(comparison), 0644)
	if err != nil {
		log.Printf("⚠️  Failed to save comparison to file: %v", err)
	} else {
		fmt.Printf("💾 Detailed comparison saved to: %s\n", outputFile)
	}
}
