package contextual

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

//nolint:containedctx // It's a context wrapper that includes errgroup.Group and context.CancelFunc.
type Cancellable struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	errg   *errgroup.Group
	// health health.Health
}

func NewCancellable(ctx context.Context) Context {
	ctx, cancel := context.WithCancelCause(ctx)
	errg, ctx := errgroup.WithContext(ctx)
	// health := health.NewCore(logmgr.Named("Health"))

	return &Cancellable{
		ctx:    ctx,
		cancel: cancel,
		errg:   errg,
		// health: health,
	}
}

func (c *Cancellable) CloneWithNewContext(ctx context.Context, cancel context.CancelCauseFunc) Context {
	return &Cancellable{
		ctx:    ctx,
		cancel: cancel,
		errg:   c.errg,
	}
}

func (c *Cancellable) ReplaceContext(cb func(context.Context) context.Context) {
	c.ctx = cb(c.ctx)
}

// func (c *Cancellable) Health() health.Health {
// 	return c.health
// }

// Cancel calls the context.CancelFunc.
// A CancelFunc tells an operation to abandon its work.
// A CancelFunc does not wait for the work to stop.
// A CancelFunc may be called by multiple goroutines simultaneously.
// After the first call, subsequent calls to a CancelFunc do nothing.
func (c *Cancellable) Cancel() {
	c.cancel(context.Canceled)
}

// CancelWithCause behaves like [Cancel] but additionally sets the cancellation cause.
// This cause can be retrieved by calling [Cause] on the canceled Context or on
// any of its derived Contexts.
//
// If the context has already been canceled, CancelCauseFunc does not set the cause.
// For example, if childContext is derived from parentContext:
//   - if parentContext is canceled with cause1 before childContext is canceled with cause2,
//     then Cause(parentContext) == Cause(childContext) == cause1
//   - if childContext is canceled with cause2 before parentContext is canceled with cause1,
//     then Cause(parentContext) == cause1 and Cause(childContext) == cause2
func (c *Cancellable) CancelWithCause(err error) {
	c.cancel(err)
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
//
//nolint:wrapcheck // transparent method to call internal context.
func (c *Cancellable) Wait() error {
	return c.errg.Wait()
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by Wait.
func (c *Cancellable) Go(f func() error) {
	c.errg.Go(f)
}

func (c *Cancellable) Deadline() (time.Time, bool) {
	return c.ctx.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
// The close of the Done channel may happen asynchronously,
// after the cancel function returns.
func (c *Cancellable) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns the context error.
// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
//
//nolint:wrapcheck // transparent method to call internal context.
func (c *Cancellable) Err() error {
	return c.ctx.Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Cancellable) Value(key any) any {
	return c.ctx.Value(key)
}
