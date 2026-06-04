//go:build !linux

package netlink

import (
	"context"
	"fmt"

	"nms-agent/internal/config"
	"nms-agent/internal/discovery"
)

type Provider struct{}

func (p Provider) Candidates(ctx context.Context, loaded config.Loaded) ([]discovery.Candidate, error) {
	_ = ctx
	_ = loaded
	return nil, fmt.Errorf("netlink discovery provider is only supported on linux")
}
