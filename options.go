package contextual

import (
	"context"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

type OptionFunc func(Context) Context

func WithPProfLabels(labelSet pprof.LabelSet) OptionFunc {
	return func(ctx Context) Context {
		ctx.ReplaceContext(func(ctx context.Context) context.Context {
			return pprof.WithLabels(ctx, labelSet)
		})

		return ctx
	}
}

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

func WithValues(args []ContextKV) OptionFunc {
	return func(ctx Context) Context {
		if valCtx, ok := ctx.(ContextValueStore); ok {
			for _, arg := range args {
				valCtx.AddValue(arg.Key, arg.Value)
			}
		}
		return ctx
	}
}
