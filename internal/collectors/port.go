package collectors

import (
	"context"

	"nms-agent/internal/models"
)

// Collector gathers raw device data (SNMP/ICMP/etc).
// Implementations live behind this contract.
type Collector interface {
	Collect(ctx context.Context) ([]models.RawSample, error)
}
