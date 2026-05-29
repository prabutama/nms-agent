package adapters

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"nms-agent/internal/models"
)

type TUIAdapter struct {
	program *tea.Program
	done    chan struct{}
}

type tuiConfig struct {
	RefreshInterval time.Duration
	AltScreen       bool
	DisableRenderer bool
	DiscardOutput   bool
	DemoMode        bool
}

func parseTUIConfig(cfg map[string]any) tuiConfig {
	c := tuiConfig{
		RefreshInterval: 1 * time.Second,
		AltScreen:       true,
		DisableRenderer: false,
		DiscardOutput:   false,
	}
	if cfg == nil {
		return c
	}
	if v, ok := cfg["refresh_interval"].(string); ok {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			c.RefreshInterval = d
		}
	}
	if v, ok := cfg["alt_screen"].(bool); ok {
		c.AltScreen = v
	}
	if v, ok := cfg["disable_renderer"].(bool); ok {
		c.DisableRenderer = v
	}
	if v, ok := cfg["discard_output"].(bool); ok {
		c.DiscardOutput = v
	}
	return c
}

func NewTUIAdapter(cfg map[string]any) (*TUIAdapter, error) {
	c := parseTUIConfig(cfg)
	m := newTUIModel(c.RefreshInterval)

	out := io.Writer(os.Stdout)
	in := io.Reader(os.Stdin)
	if c.DiscardOutput {
		out = io.Discard
		in = nil
	}

	opts := []tea.ProgramOption{tea.WithOutput(out), tea.WithInput(in), tea.WithoutSignals()}
	if c.AltScreen {
		opts = append(opts, tea.WithAltScreen())
	}
	if c.DisableRenderer {
		opts = append(opts, tea.WithoutRenderer())
	}

	p := tea.NewProgram(m, opts...)
	done := make(chan struct{})
	a := &TUIAdapter{program: p, done: done}

	go func() {
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		}
		close(done)
	}()

	return a, nil
}

func (a *TUIAdapter) SendBatch(_ context.Context, batch []models.Telemetry) error {
	select {
	case <-a.done:
		return nil
	default:
		a.program.Send(telemetryBatchMsg(batch))
		return nil
	}
}

func (a *TUIAdapter) Close() error {
	if a == nil || a.program == nil {
		return nil
	}
	a.program.Quit()
	select {
	case <-a.done:
	case <-time.After(2 * time.Second):
	}
	return nil
}

// For tests, allow overriding Bubble Tea output.
func newTUIProgramForTest(m tea.Model, out io.Writer) *tea.Program {
	if out == nil {
		out = io.Discard
	}
	return tea.NewProgram(m, tea.WithOutput(out), tea.WithoutRenderer(), tea.WithoutSignals())
}
