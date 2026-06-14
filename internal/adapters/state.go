package adapters

import (
	"nms-agent/internal/adapters/base"
	"nms-agent/internal/models"
)

type State = base.State
type DeviceState = base.DeviceState
type IfaceState = base.IfaceState
type AlertState = base.AlertState
type DeviceResources = base.DeviceResources
type StorageState = base.StorageState

func NewState() *State {
	return base.NewState()
}

func NewStateFromTelemetry(batch []models.Telemetry) *State {
	return base.NewStateFromTelemetry(batch)
}
