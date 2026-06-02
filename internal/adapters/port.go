package adapters

import (
	"context"

	"nms-agent/internal/models"
)

// Adapter sends telemetry to a consumer platform.
// All platform-specific behavior must be implemented behind this contract.
type Adapter interface {
	SendBatch(ctx context.Context, batch []models.Telemetry) error
}

// HealthChecker is an optional interface implemented by adapters that can
// proactively verify connectivity (without sending telemetry).
//
// This is used by tooling (e.g., nms-agentctl) and must not modify queue state.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// Closable is an optional interface implemented by adapters that hold resources.
type Closable interface {
	Close() error
}

// ObserverSetter allows attaching a viewer hub for local viewing.
type ObserverSetter interface {
	SetObserver(hub AdapterObserver)
}

// AdapterObserver receives telemetry batches and status updates for local viewing.
type AdapterObserver interface {
	Update(batch []models.Telemetry)
	UpdateStatus(status string, details string)
}
