package selector

import (
	"context"
)

type (
	consistHash struct{}
	roundRobin  struct{}
	specifyAddr struct{}
)

func WithConsistHash(ctx context.Context, hashKey string) context.Context {
	return with(ctx, consistHash{}, hashKey)
}

func WithRoundRobin(ctx context.Context) context.Context {
	return with(ctx, roundRobin{}, struct{}{})
}

func WithSpecifyAddr(ctx context.Context, addr string) context.Context {
	return with(ctx, specifyAddr{}, addr)
}

func with(ctx context.Context, key, value interface{}) context.Context {
	if ctx == nil {
		ctx = context.TODO()
	}
	return context.WithValue(ctx, key, value)
}
