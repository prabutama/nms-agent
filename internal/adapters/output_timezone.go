package adapters

import (
	"time"

	"nms-agent/internal/adapters/base"
)

func SetOutputLocation(loc *time.Location) {
	base.SetOutputLocation(loc)
}

func GetOutputLocation() *time.Location {
	return base.GetOutputLocation()
}
