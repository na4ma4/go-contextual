package contextual

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

type OptionFunc func(Context) Context

func WithTimeoutOption(timeout time.Duration) OptionFunc {
	return func(ctx Context) Context {
		ctx, _ = WithTimeout(ctx, timeout)
		return ctx
	}
}

func WithDeadlineOption(deadline time.Time) OptionFunc {
	return func(ctx Context) Context {
		ctx, _ = WithDeadline(ctx, deadline)
		return ctx
	}
}

func WithSignalCancelOption(signals ...os.Signal) OptionFunc {
	return func(ctx Context) Context {
		if len(signals) == 0 {
			signals = []os.Signal{syscall.SIGTERM, syscall.SIGINT}
		}

		rawCtx, cancel := signal.NotifyContext(ctx, signals...)
		rootCtx := ctx.CloneWithNewContext(rawCtx, CancelCauseWrap(cancel))
		return rootCtx
	}
}
