package postgres

import "context"

type PoolPinger struct {
	PingFn func(ctx context.Context) error
}

func (p PoolPinger) Ping(ctx context.Context) error {
	return p.PingFn(ctx)
}
