package sensor

import "peerdrive/app/pkg/models"

type Manager interface {
	StartTracking() (<-chan models.VehicleState, error)
	StopTracking()
}