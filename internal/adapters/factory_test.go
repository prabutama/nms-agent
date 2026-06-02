package adapters

import "testing"

func TestNewAdapter_SupportedNames(t *testing.T) {
	// tui (headless)
	ad, err := NewAdapter("tui", map[string]any{"alt_screen": false, "discard_output": true, "disable_renderer": true})
	if err != nil {
		t.Fatalf("tui: %v", err)
	}
	if c, ok := ad.(Closable); ok {
		_ = c.Close()
	}

	// generic mqtt
	ad, err = NewAdapter("generic_mqtt", map[string]any{"broker": "tcp://127.0.0.1:1883", "topic": "t"})
	if err != nil {
		t.Fatalf("generic_mqtt: %v", err)
	}
	if c, ok := ad.(Closable); ok {
		_ = c.Close()
	}

	// thingsboard mqtt
	ad, err = NewAdapter("thingsboard_mqtt", map[string]any{"broker": "tcp://127.0.0.1:1883", "access_token": "token"})
	if err != nil {
		t.Fatalf("thingsboard_mqtt: %v", err)
	}
	if c, ok := ad.(Closable); ok {
		_ = c.Close()
	}
}

func TestNewAdapter_UnknownNameFails(t *testing.T) {
	if _, err := NewAdapter("nope", nil); err == nil {
		t.Fatalf("expected error")
	}
}
