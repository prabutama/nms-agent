package models

import (
	"encoding/json"
	"time"
)

// RawSample is unnormalized data as collected from devices.
// It intentionally stays generic and platform-agnostic.
type RawSample struct {
	DeviceID string
	Source   string // e.g. "snmp", "icmp" (collector-defined)
	TS       time.Time
	Fields   map[string]any
}

// Telemetry is the canonical, normalized record persisted to the local queue.
type Telemetry struct {
	DeviceID    string
	Metric      string
	TS          time.Time
	ValueType   string
	ValueNumber *float64
	ValueString *string
	Tags        map[string]string
}

type telemetryJSON struct {
	DeviceID    string            `json:"DeviceID"`
	Metric      string            `json:"Metric"`
	TS          time.Time         `json:"TS"`
	ValueType   string            `json:"ValueType,omitempty"`
	ValueNumber *float64          `json:"ValueNumber,omitempty"`
	ValueString *string           `json:"ValueString,omitempty"`
	Value       *float64          `json:"Value,omitempty"`
	Tags        map[string]string `json:"Tags,omitempty"`
}

func (t Telemetry) MarshalJSON() ([]byte, error) {
	return json.Marshal(telemetryJSON{
		DeviceID:    t.DeviceID,
		Metric:      t.Metric,
		TS:          t.TS,
		ValueType:   t.ValueType,
		ValueNumber: t.ValueNumber,
		ValueString: t.ValueString,
		Tags:        t.Tags,
	})
}

func (t *Telemetry) UnmarshalJSON(data []byte) error {
	var tmp telemetryJSON
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if tmp.ValueType == "" {
		switch {
		case tmp.ValueString != nil:
			tmp.ValueType = "string"
		case tmp.ValueNumber != nil:
			tmp.ValueType = "number"
		case tmp.Value != nil:
			tmp.ValueType = "number"
			tmp.ValueNumber = tmp.Value
		}
	}
	*t = Telemetry{
		DeviceID:    tmp.DeviceID,
		Metric:      tmp.Metric,
		TS:          tmp.TS,
		ValueType:   tmp.ValueType,
		ValueNumber: tmp.ValueNumber,
		ValueString: tmp.ValueString,
		Tags:        tmp.Tags,
	}
	return nil
}
