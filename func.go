package contextual

import (
	"context"
	"os/signal"
	"syscall"
)

func WithCancel(ctx *CtxCancel) (*CtxCancel, context.CancelFunc) {
	rawCtx, cancel := context.WithCancel(ctx)
	return &CtxCancel{
		ctx:    rawCtx,
		cancel: cancel,
		errg:   ctx.errg,
		health: ctx.health,
	}, cancel
}

func WithSignalCancel(ctx *CtxCancel) (*CtxCancel, context.CancelFunc) {
	rawCtx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	return &CtxCancel{
		ctx:    rawCtx,
		cancel: cancel,
		errg:   ctx.errg,
		health: ctx.health,
	}, cancel
}
