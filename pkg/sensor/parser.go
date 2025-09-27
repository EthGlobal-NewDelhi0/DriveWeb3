package sensor

import (
	"encoding/json"
	"peerdrive/app/pkg/models"
)

// ParseDataToVehicleState converts a raw byte slice from hardware into a
// structured VehicleState.
//
// NOTE: This is a placeholder implementation. You will need to replace this
// with the actual parsing logic based on the data format your hardware dongle
// sends (e.g., NMEA, custom binary, etc.).
//
// For now, we assume the hardware sends data as a JSON string.
func ParseDataToVehicleState(data []byte) (models.VehicleState, error) {      // TODO: implement this for real data
	var state models.VehicleState
	err := json.Unmarshal(data, &state)
	return state, err
}