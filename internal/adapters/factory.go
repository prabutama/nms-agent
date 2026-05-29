package adapters

import "fmt"

// NewAdapter returns an adapter implementation selected by name.
// config is adapter-specific (e.g., TUI options).
func NewAdapter(name string, config map[string]any) (Adapter, error) {
	switch name {
	case "terminal":
		return NewTerminalAdapter(), nil
	case "tui":
		return NewTUIAdapter(config)
	default:
		return nil, fmt.Errorf("unknown adapter %q (supported: terminal, tui)", name)
	}
}
