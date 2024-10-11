package contextual

import "runtime/pprof"

func SetLabelsFromContext(ctx Context) {
	pprof.SetGoroutineLabels(ctx)
}
