package viewer

import (
	"time"

	"nms-agent/internal/models"
)

// Message is a line-delimited JSON payload for viewer clients.
// Type can be "snapshot", "telemetry", or "status".
type Message struct {
	Type      string             `json:"type"`
	Adapter   string             `json:"adapter,omitempty"`
	Telemetry []models.Telemetry `json:"telemetry,omitempty"`
	Status    string             `json:"status,omitempty"`
	Details   string             `json:"details,omitempty"`
	At        time.Time          `json:"at"`
}
