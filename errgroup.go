package contextual

import "context"

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
