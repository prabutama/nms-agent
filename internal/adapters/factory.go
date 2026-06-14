package adapters

import (
	"fmt"

	"nms-agent/internal/adapters/generic_mqtt"
	"nms-agent/internal/adapters/thingsboard_mqtt"
	"nms-agent/internal/adapters/tui"
)

func NewAdapter(name string, config map[string]any) (Adapter, error) {
	switch name {
	case "tui":
		return tui.NewAdapter(config)
	case "generic_mqtt":
		return genericmqtt.NewAdapter(config)
	case "thingsboard_mqtt":
		return thingsboardmqtt.NewAdapter(config)
	default:
		return nil, fmt.Errorf("unknown adapter %q (supported: tui, generic_mqtt, thingsboard_mqtt)", name)
	}
}
