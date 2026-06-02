package adapters

import "fmt"

// NewAdapter returns an adapter implementation selected by name.
// config is adapter-specific (e.g., TUI options).
func NewAdapter(name string, config map[string]any) (Adapter, error) {
	switch name {
	case "tui":
		return NewTUIAdapter(config)
	case "generic_mqtt":
		return NewGenericMQTTAdapter(config)
	case "thingsboard_mqtt":
		return NewThingsBoardMQTTAdapter(config)
	default:
		return nil, fmt.Errorf("unknown adapter %q (supported: tui, generic_mqtt, thingsboard_mqtt)", name)
	}
}
