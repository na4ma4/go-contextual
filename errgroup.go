package contextual

import (
	"context"
	"runtime/pprof"
)

type FuncErr func() error
type CtxErrFunc func(context.Context) error
type CtxualErrFunc func(Context) error

type errgroupFuncs interface {
	FuncErr | CtxErrFunc | CtxualErrFunc
}

func Go[T errgroupFuncs](ctx Context, f T) {
	switch v := any(f).(type) {
	case FuncErr:
		ctx.Go(v)
	case CtxErrFunc:
		ctx.Go(func() error {
			return v(ctx)
		})
	case CtxualErrFunc:
		ctx.Go(func() error {
			return v(ctx)
		})
	default:
		panic("contextual.Go() generic with unknown type")
	}
}

func GoLabelled[T errgroupFuncs](ctx Context, name, description string, f T) {
	labelSet := CommonLabels(name, description)
	switch v := any(f).(type) {
	case FuncErr:
		ctx.GoLabelled(labelSet, v)
	case CtxErrFunc:
		ctx.GoLabelled(labelSet, func() error {
			return v(ctx)
		})
	case CtxualErrFunc:
		ctx.GoLabelled(labelSet, func() error {
			return v(ctx)
		})
	default:
		panic("contextual.Go() generic with unknown type")
	}
}

func CommonLabels(name, description string) pprof.LabelSet {
	return pprof.Labels("name", name, "description", description)
}

func Labels(args ...string) pprof.LabelSet {
	return pprof.Labels(args...)
}
