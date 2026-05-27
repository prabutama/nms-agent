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

		valueType, _ := r.Fields["value_type"].(string)
		var valueNumber *float64
		var valueString *string
		if valueType == "" {
			return nil, fmt.Errorf("raw sample missing field 'value_type'")
		}
		switch valueType {
		case "number":
			v, ok := r.Fields["value_number"]
			if !ok {
				return nil, fmt.Errorf("raw sample missing field 'value_number'")
			}
			if num, ok := v.(float64); ok {
				valueNumber = &num
			} else if iv, ok := v.(int); ok {
				num := float64(iv)
				valueNumber = &num
			} else {
				return nil, fmt.Errorf("raw sample field 'value_number' must be number")
			}
		case "string":
			v, ok := r.Fields["value_string"]
			if !ok {
				return nil, fmt.Errorf("raw sample missing field 'value_string'")
			}
			if s, ok := v.(string); ok {
				valueString = &s
			} else {
				return nil, fmt.Errorf("raw sample field 'value_string' must be string")
			}
		default:
			return nil, fmt.Errorf("raw sample field 'value_type' must be number|string")
		}

		tags := map[string]string{}
		if unit, _ := r.Fields["unit"].(string); unit != "" {
			tags["unit"] = unit
		}
		if extra, ok := r.Fields["tags"].(map[string]string); ok {
			for k, v := range extra {
				if k == "" || v == "" {
					continue
				}
				tags[k] = v
			}
		}
		if r.Source != "" {
			tags["source"] = r.Source
		}

		out = append(out, models.Telemetry{
			DeviceID:    r.DeviceID,
			Metric:      metric,
			TS:          r.TS,
			ValueType:   valueType,
			ValueNumber: valueNumber,
			ValueString: valueString,
			Tags:        tags,
		})
	}
	return out, nil
}
