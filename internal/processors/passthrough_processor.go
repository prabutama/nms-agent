package processors

import (
	"context"
	"fmt"

	"nms-agent/internal/models"
)

// PassthroughProcessor maps dummy RawSample fields into canonical Telemetry.
// It does not do real preprocessing/normalization yet.
type PassthroughProcessor struct{}

func (PassthroughProcessor) Normalize(ctx context.Context, raw []models.RawSample) ([]models.Telemetry, error) {
	out := make([]models.Telemetry, 0, len(raw))
	for _, r := range raw {
		metric, _ := r.Fields["metric"].(string)
		if metric == "" {
			metric = "unknown"
		}

		v, ok := r.Fields["value"]
		if !ok {
			return nil, fmt.Errorf("raw sample missing field 'value'")
		}
		value, ok := v.(float64)
		if !ok {
			// Accept ints for convenience in tests/demo.
			if iv, ok := v.(int); ok {
				value = float64(iv)
			} else {
				return nil, fmt.Errorf("raw sample field 'value' must be number")
			}
		}

		tags := map[string]string{}
		if unit, _ := r.Fields["unit"].(string); unit != "" {
			tags["unit"] = unit
		}
		if r.Source != "" {
			tags["source"] = r.Source
		}

		out = append(out, models.Telemetry{
			DeviceID: r.DeviceID,
			Metric:   metric,
			TS:       r.TS,
			Value:    value,
			Tags:     tags,
		})
	}
	return out, nil
}
