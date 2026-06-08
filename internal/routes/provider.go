package routes

import "context"

type Provider interface {
	Collect(ctx context.Context, deviceID, address string) (RouteSnapshot, error)
}
