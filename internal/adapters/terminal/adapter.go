package terminal

import (
	"context"
	"io"
	"os"

	"nms-agent/internal/adapters/base"
	"nms-agent/internal/models"
)

type TerminalAdapter struct {
	Out io.Writer
	obs base.AdapterObserver
}

func New() *TerminalAdapter {
	return &TerminalAdapter{Out: os.Stdout}
}

func (a *TerminalAdapter) SetObserver(hub base.AdapterObserver) {
	a.obs = hub
}

func (a *TerminalAdapter) SendBatch(ctx context.Context, batch []models.Telemetry) error {
	_ = ctx
	if a.Out == nil {
		a.Out = os.Stdout
	}
	for _, t := range batch {
		base.FormatTS(t.TS)
	}
	if a.obs != nil {
		a.obs.Update(batch)
	}
	return nil
}
