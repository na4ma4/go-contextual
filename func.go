package contextual

import (
	"context"
	"os/signal"
	"syscall"
	"time"
)

// WithTimeout returns WithDeadline(parent, time.Now().Add(timeout)).
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this [Context] complete:
//
//	func slowOperationWithTimeout(ctx context.Context) (Result, error) {
//		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
//		defer cancel()  // releases resources if slowOperation completes before timeout elapses
//		return slowOperation(ctx)
//	}
func WithTimeout(parent context.Context, timeout time.Duration) (Context, context.CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

// WithTimeoutCause behaves like [WithTimeout] but also sets the cause of the
// returned Context when the timeout expires. The returned [CancelFunc] does
// not set the cause.
func WithTimeoutCause(parent context.Context, timeout time.Duration, cause error) (Context, context.CancelFunc) {
	return WithDeadlineCause(parent, time.Now().Add(timeout), cause)
}

// WithDeadline returns a copy of the parent context with the deadline adjusted
// to be no later than d. If the parent's deadline is already earlier than d,
// WithDeadline(parent, d) is semantically equivalent to parent. The returned
// [Context.Done] channel is closed when the deadline expires, when the returned
// cancel function is called, or when the parent context's Done channel is
// closed, whichever happens first.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this [Context] complete.
func WithDeadline(parent context.Context, d time.Time) (Context, context.CancelFunc) {
	return WithDeadlineCause(parent, d, nil)
}

// WithDeadlineCause behaves like [WithDeadline] but also sets the cause of the
// returned Context when the deadline is exceeded. The returned [CancelFunc] does
// not set the cause.
func WithDeadlineCause(parent context.Context, d time.Time, cause error) (Context, context.CancelFunc) {
	rootCtx, cancel := context.WithDeadlineCause(parent, d, cause)
	var outCtx Context
	if parentIsCtxl, ok := parent.(Context); ok {
		outCtx = parentIsCtxl.CloneWithNewContext(rootCtx, CancelCauseWrap(cancel))
	} else {
		outCtx = NewCancellable(rootCtx)
	}

	return outCtx, outCtx.Cancel
}

// WithCancel returns a copy of parent with a new Done channel. The returned
// context's Done channel is closed when the returned cancel function is called
// or when the parent context's Done channel is closed, whichever happens first.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func WithCancel(ctx Context) (Context, context.CancelFunc) {
	rootCtx := ctx.CloneWithNewContext(context.WithCancelCause(ctx))
	return rootCtx, rootCtx.Cancel
}

// WithCancelCause behaves like [WithCancel] but returns a [CancelCauseFunc] instead of a [CancelFunc].
// Calling cancel with a non-nil error (the "cause") records that error in ctx;
// it can then be retrieved using Cause(ctx).
// Calling cancel with nil sets the cause to Canceled.
//
// Example use:
//
//	ctx, cancel := context.WithCancelCause(parent)
//	cancel(myError)
//	ctx.Err() // returns context.Canceled
//	context.Cause(ctx) // returns myError
func WithCancelCause(ctx Context) (Context, context.CancelCauseFunc) {
	rootCtx := ctx.CloneWithNewContext(context.WithCancelCause(ctx))
	return rootCtx, rootCtx.CancelWithCause
}

// NotifyContext returns a copy of the parent context that is marked done
// (its Done channel is closed) when one of the listed signals arrives,
// when the returned stop function is called, or when the parent context's
// Done channel is closed, whichever happens first.
//
// The stop function unregisters the signal behavior, which, like signal.Reset,
// may restore the default behavior for a given signal. For example, the default
// behavior of a Go program receiving os.Interrupt is to exit. Calling
// NotifyContext(parent, os.Interrupt) will change the behavior to cancel
// the returned context. Future interrupts received will not trigger the default
// (exit) behavior until the returned stop function is called.
//
// The stop function releases resources associated with it, so code should
// call stop as soon as the operations running in this Context complete and
// signals no longer need to be diverted to the context.
func WithSignalCancel(ctx Context) (Context, context.CancelFunc) {
	rawCtx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	rootCtx := ctx.CloneWithNewContext(rawCtx, CancelCauseWrap(cancel))
	return rootCtx, rootCtx.Cancel
}

func CancelCauseWrap(cancel context.CancelFunc) context.CancelCauseFunc {
	return func(_ error) {
		cancel()
	}
}
