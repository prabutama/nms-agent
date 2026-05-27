package collectors

import (
	"context"
	"time"

	"nms-agent/internal/models"
)

// DummyCollector generates deterministic samples for demo/testing.
// It performs no network calls.
type DummyCollector struct {
	DeviceID string
}

func (c DummyCollector) Collect(context.Context) ([]models.RawSample, error) {
	deviceID := c.DeviceID
	if deviceID == "" {
		deviceID = "dummy-1"
	}

	now := time.Now().UTC()
	return []models.RawSample{
		{
			DeviceID: deviceID,
			Source:   "dummy",
			TS:       now,
			Fields: map[string]any{
				"metric":       "demo.ping",
				"value_type":   "number",
				"value_number": 42.0,
				"unit":         "ms",
			},
		},
	}, nil
}
