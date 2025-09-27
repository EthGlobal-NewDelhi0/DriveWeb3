package strategy

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// ScenarioReport contains a complete report of a scenario run
type ScenarioReport struct {
	ScenarioType      ScenarioType      `json:"scenarioType"`
	CommunicationMode CommunicationMode `json:"communicationMode"`
	Timestamp         time.Time         `json:"timestamp"`
	Duration          time.Duration     `json:"duration"`
	Metrics           []ScenarioMetrics `json:"metrics"`
	Summary           ScenarioSummary   `json:"summary"`
}

// ScenarioSummary provides aggregated statistics for a scenario
type ScenarioSummary struct {
	TotalVehicles        int     `json:"totalVehicles"`
	CollisionCount       int     `json:"collisionCount"`
	AvgTimeToCollision   float64 `json:"avgTimeToCollision"`
	AvgPeakDeceleration  float64 `json:"avgPeakDeceleration"`
	AvgV2VResponseTime   float64 `json:"avgV2VResponseTime"`
	EmergencyPassageTime float64 `json:"emergencyPassageTime"`
	SafetyScore          float64 `json:"safetyScore"` // 0-100 score based on multiple factors
}

// SaveMetricsToJSON saves scenario metrics to a JSON file
func SaveMetricsToJSON(metrics []ScenarioMetrics, scenarioType ScenarioType, commMode CommunicationMode, duration time.Duration, filename string) error {
	summary := calculateSummary(metrics)

	report := ScenarioReport{
		ScenarioType:      scenarioType,
		CommunicationMode: commMode,
		Timestamp:         time.Now(),
		Duration:          duration,
		Metrics:           metrics,
		Summary:           summary,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode metrics: %v", err)
	}

	return nil
}

// SaveMetricsToCSV saves scenario metrics to a CSV file for analysis
func SaveMetricsToCSV(metrics []ScenarioMetrics, scenarioType ScenarioType, commMode CommunicationMode, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %v", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"ScenarioType", "CommunicationMode", "VehicleID", "TimeToCollision",
		"PeakDeceleration", "SpacingError", "CollisionOccurred",
		"TimeToRegainSpeed", "V2VResponseTime", "FinalPosition", "EmergencyPassageTime",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %v", err)
	}

	// Write data rows
	scenarioName := getScenarioTypeName(scenarioType)
	commModeName := getCommunicationModeName(commMode)

	for _, metric := range metrics {
		row := []string{
			scenarioName,
			commModeName,
			metric.VehicleID,
			fmt.Sprintf("%.6f", metric.TimeToCollision),
			fmt.Sprintf("%.6f", metric.PeakDeceleration),
			fmt.Sprintf("%.6f", metric.SpacingError),
			strconv.FormatBool(metric.CollisionOccurred),
			fmt.Sprintf("%.6f", metric.TimeToRegainSpeed),
			fmt.Sprintf("%.6f", metric.V2VResponseTime),
			fmt.Sprintf("%.6f", metric.FinalPosition),
			fmt.Sprintf("%.6f", metric.EmergencyPassageTime),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %v", err)
		}
	}

	return nil
}

// LoadMetricsFromJSON loads scenario metrics from a JSON file
func LoadMetricsFromJSON(filename string) (*ScenarioReport, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	var report ScenarioReport
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode metrics: %v", err)
	}

	return &report, nil
}

// CompareScenarios compares metrics between two scenarios and returns a comparison report
func CompareScenarios(report1, report2 *ScenarioReport) string {
	comparison := "# Scenario Comparison Report\n\n"

	comparison += fmt.Sprintf("## Scenario 1: %s (%s)\n",
		getScenarioTypeName(report1.ScenarioType),
		getCommunicationModeName(report1.CommunicationMode))
	comparison += fmt.Sprintf("- Vehicles: %d\n", report1.Summary.TotalVehicles)
	comparison += fmt.Sprintf("- Collisions: %d\n", report1.Summary.CollisionCount)
	comparison += fmt.Sprintf("- Avg TTC: %.2fs\n", report1.Summary.AvgTimeToCollision)
	comparison += fmt.Sprintf("- Avg Peak Deceleration: %.2f m/s²\n", report1.Summary.AvgPeakDeceleration)
	comparison += fmt.Sprintf("- Avg V2V Response: %.3fs\n", report1.Summary.AvgV2VResponseTime)
	comparison += fmt.Sprintf("- Safety Score: %.1f/100\n\n", report1.Summary.SafetyScore)

	comparison += fmt.Sprintf("## Scenario 2: %s (%s)\n",
		getScenarioTypeName(report2.ScenarioType),
		getCommunicationModeName(report2.CommunicationMode))
	comparison += fmt.Sprintf("- Vehicles: %d\n", report2.Summary.TotalVehicles)
	comparison += fmt.Sprintf("- Collisions: %d\n", report2.Summary.CollisionCount)
	comparison += fmt.Sprintf("- Avg TTC: %.2fs\n", report2.Summary.AvgTimeToCollision)
	comparison += fmt.Sprintf("- Avg Peak Deceleration: %.2f m/s²\n", report2.Summary.AvgPeakDeceleration)
	comparison += fmt.Sprintf("- Avg V2V Response: %.3fs\n", report2.Summary.AvgV2VResponseTime)
	comparison += fmt.Sprintf("- Safety Score: %.1f/100\n\n", report2.Summary.SafetyScore)

	// Calculate improvements
	collision_improvement := report1.Summary.CollisionCount - report2.Summary.CollisionCount
	ttc_improvement := report2.Summary.AvgTimeToCollision - report1.Summary.AvgTimeToCollision
	safety_improvement := report2.Summary.SafetyScore - report1.Summary.SafetyScore

	comparison += "## Key Differences:\n"
	comparison += fmt.Sprintf("- Collision reduction: %d fewer collisions\n", collision_improvement)
	comparison += fmt.Sprintf("- TTC improvement: %.2fs longer average time to collision\n", ttc_improvement)
	comparison += fmt.Sprintf("- Safety improvement: %.1f point increase\n", safety_improvement)

	if report2.Summary.AvgV2VResponseTime > 0 && report1.Summary.AvgV2VResponseTime == 0 {
		comparison += fmt.Sprintf("- V2V communication enabled with %.3fs average response time\n", report2.Summary.AvgV2VResponseTime)
	}

	return comparison
}

// calculateSummary computes aggregated statistics from individual vehicle metrics
func calculateSummary(metrics []ScenarioMetrics) ScenarioSummary {
	if len(metrics) == 0 {
		return ScenarioSummary{}
	}

	summary := ScenarioSummary{
		TotalVehicles: len(metrics),
	}

	var ttcSum, decelSum, v2vSum float64
	var ttcCount, v2vCount int

	for _, metric := range metrics {
		// Count collisions
		if metric.CollisionOccurred {
			summary.CollisionCount++
		}

		// Sum time to collision (exclude infinite values)
		if metric.TimeToCollision < 1000 {
			ttcSum += metric.TimeToCollision
			ttcCount++
		}

		// Sum peak deceleration
		decelSum += metric.PeakDeceleration

		// Sum V2V response times (exclude zero values)
		if metric.V2VResponseTime > 0 {
			v2vSum += metric.V2VResponseTime
			v2vCount++
		}

		// Emergency passage time (for ambulance)
		if metric.EmergencyPassageTime > 0 {
			summary.EmergencyPassageTime = metric.EmergencyPassageTime
		}
	}

	// Calculate averages
	if ttcCount > 0 {
		summary.AvgTimeToCollision = ttcSum / float64(ttcCount)
	}
	summary.AvgPeakDeceleration = decelSum / float64(len(metrics))
	if v2vCount > 0 {
		summary.AvgV2VResponseTime = v2vSum / float64(v2vCount)
	}

	// Calculate safety score (0-100)
	// Based on: collision rate, average TTC, and peak deceleration
	safetyScore := 100.0

	// Penalize collisions heavily
	collisionRate := float64(summary.CollisionCount) / float64(summary.TotalVehicles)
	safetyScore -= collisionRate * 50 // Up to -50 points for collisions

	// Penalize low TTC
	if summary.AvgTimeToCollision < 5.0 && ttcCount > 0 {
		safetyScore -= (5.0 - summary.AvgTimeToCollision) * 5 // Up to -25 points for low TTC
	}

	// Penalize high deceleration
	if summary.AvgPeakDeceleration > 6.0 {
		safetyScore -= (summary.AvgPeakDeceleration - 6.0) * 5 // Penalty for harsh braking
	}

	// Bonus for V2V communication
	if summary.AvgV2VResponseTime > 0 && summary.AvgV2VResponseTime < 0.1 {
		safetyScore += 10 // Bonus for fast V2V response
	}

	if safetyScore < 0 {
		safetyScore = 0
	}
	summary.SafetyScore = safetyScore

	return summary
}

// Helper functions for readable names
func getScenarioTypeName(scenarioType ScenarioType) string {
	switch scenarioType {
	case ScenarioFiveVehicleStopping:
		return "FiveVehicleStopping"
	case ScenarioEmergencyVehicle:
		return "EmergencyVehicle"
	default:
		return "Unknown"
	}
}

func getCommunicationModeName(commMode CommunicationMode) string {
	switch commMode {
	case NoComm:
		return "NoComm"
	case V2VComm:
		return "V2VComm"
	default:
		return "Unknown"
	}
}

// GenerateTimestampedFilename creates a filename with timestamp
func GenerateTimestampedFilename(prefix, extension string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s.%s", prefix, timestamp, extension)
}
