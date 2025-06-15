package contextual

import (
	"context"
	"runtime/pprof"
)

// FuncErr defines a function signature for operations that can be run by an errgroup.
// It takes no arguments and returns an error.
// Use this type for goroutines managed by `Go` or `GoLabelled` that do not
// require access to a context.
type FuncErr func() error

// CtxErrFunc defines a function signature for operations that can be run by an errgroup
// and require a standard Go context.
// It takes a context.Context as an argument and returns an error.
// Use this type for goroutines managed by `Go` or `GoLabelled` when the task needs
// to be aware of cancellation or deadlines via a standard `context.Context`,
// but does not need the specific features of `contextual.Context`.
type CtxErrFunc func(context.Context) error

// CtxualErrFunc defines a function signature for operations that can be run by an errgroup
// and require a contextual.Context.
// It takes a contextual.Context as an argument and returns an error.
// Use this type for goroutines managed by `Go` or `GoLabelled` when the task needs
// access to the enhanced features of `contextual.Context`, such as launching further
// goroutines with `ctx.Go`, accessing the value store, or other specific functionalities
// of this library.
type CtxualErrFunc func(Context) error

// errgroupFuncs is a type constraint that unions the function types supported by Go and GoLabelled.
type errgroupFuncs interface {
	FuncErr | CtxErrFunc | CtxualErrFunc
}

// dispatchGoFunc prepares a nil-accepting function for the underlying Go execution methods.
// It handles the different function signatures accepted by Go and GoLabelled.
// If f is nil (any typed nil like (FuncErr)(nil)), it returns nil.
func dispatchGoFunc[T errgroupFuncs](ctx Context, f T) func() error {
	if f == nil {
		return nil // The underlying Group.Go() methods or Context.GoLabelled handle nil properly (panic).
	}
	switch fn := any(f).(type) {
	case FuncErr:
		return fn
	case CtxErrFunc:
		// ctx (which is contextual.Context) implements context.Context,
		// so it can be passed directly to a function expecting context.Context.
		return func() error { return fn(ctx) }
	case CtxualErrFunc:
		return func() error { return fn(ctx) }
	default:
		// This case should ideally be unreachable due to generic constraints
		// and the type switch covering all members of the errgroupFuncs union.
		// However, including a panic for robustness against future changes.
		panic("contextual: Go/GoLabelled called with an unexpected function type")
	}
}

// Go runs the function f in a new goroutine managed by the Context's error group.
// f must match one of the signatures defined by FuncErr, CtxErrFunc, or CtxualErrFunc.
// Errors returned by f are propagated to ctx.Wait().
// If f is a typed nil (e.g., (FuncErr)(nil)), this function will cause a panic
// when ctx.Go is called with nil, which is the standard errgroup behavior.
func Go[T errgroupFuncs](ctx Context, f T) {
	wrappedF := dispatchGoFunc(ctx, f)
	ctx.Go(wrappedF)
}

// GoLabelled runs the function f in a new goroutine managed by the Context's error group,
// applying pprof labels for profiling.
// f must match one of the signatures defined by FuncErr, CtxErrFunc, or CtxualErrFunc.
// Errors returned by f are propagated to ctx.Wait().
// If f is a typed nil, this function will cause a panic when ctx.GoLabelled
// is called with a nil function, matching the behavior of (*Cancellable).GoLabelled.
func GoLabelled[T errgroupFuncs](ctx Context, name, description string, f T) {
	wrappedF := dispatchGoFunc(ctx, f)
	labelSet := CommonLabels(name, description)
	ctx.GoLabelled(labelSet, wrappedF)
}

// CommonLabels is a utility function that creates a pprof.LabelSet with two labels:
// "name" set to the provided `name` string, and "description" set to the `description` string.
// This is a common pattern used for labeling goroutines for profiling.
func CommonLabels(name, description string) pprof.LabelSet {
	return pprof.Labels("name", name, "description", description)
}

// Labels is a convenience wrapper around `pprof.Labels`.
// Example: Labels("key1", "value1", "key2", "value2")
// It takes a variadic list of strings `args` (key-value pairs) and returns a pprof.LabelSet.
// This allows for creating custom label sets with more or different labels than CommonLabels.
func Labels(args ...string) pprof.LabelSet {
	return pprof.Labels(args...)
}
