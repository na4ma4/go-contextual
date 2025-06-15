package contextual_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/na4ma4/go-contextual"
)

func TestContextWithDeadlineCause(t *testing.T) {
	testError := errors.New("test error for deadline")

	{ // deadline exceeded
		ctx, _ := contextual.WithDeadlineCause(context.Background(), time.Now().Add(time.Millisecond), testError)
		<-ctx.Done()
		if v := context.Cause(ctx); !errors.Is(v, testError) {
			t.Errorf("WithDeadlineCause(): err got '%s', want '%s'", v, testError)
		}
	}

	{ // returned cancel called
		ctx, cancel := contextual.WithDeadlineCause(context.Background(), time.Now().Add(time.Second), testError)
		cancel()
		<-ctx.Done()
		if v := context.Cause(ctx); !errors.Is(v, context.Canceled) {
			t.Errorf("WithDeadlineCause(): err got '%s', want '%s'", v, context.DeadlineExceeded)
		}
	}

	{ // contextual cancel called
		ctx, _ := contextual.WithDeadlineCause(context.Background(), time.Now().Add(time.Second), testError)
		ctx.Cancel()
		<-ctx.Done()
		if v := context.Cause(ctx); !errors.Is(v, context.Canceled) {
			t.Errorf("WithDeadlineCause(): err got '%s', want '%s'", v, context.DeadlineExceeded)
		}
	}

	{ // parent context cancel called
		rootCtx, cancel := context.WithCancel(context.Background())
		ctx, _ := contextual.WithDeadlineCause(rootCtx, time.Now().Add(time.Second), testError)
		cancel()
		<-ctx.Done()
		if v := context.Cause(ctx); !errors.Is(v, context.Canceled) {
			t.Errorf("WithDeadlineCause(): err got '%s', want '%s'", v, context.DeadlineExceeded)
		}
	}
}

func TestContextWithTimeout(t *testing.T) {
	t.Run("timeout_exceeded", func(t *testing.T) {
		parent := contextual.New(context.Background())
		defer parent.Cancel()

		ctx, cancel := contextual.WithTimeout(parent, 1*time.Millisecond)
		defer cancel()

		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Errorf("WithTimeout() error = %v, want %v", ctx.Err(), context.DeadlineExceeded)
		}
		// Check cause if possible (Go 1.20+)
		if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, context.DeadlineExceeded) {
			t.Errorf("WithTimeout() cause = %v, want %v", cause, context.DeadlineExceeded)
		}
	})

	t.Run("cancel_func_called", func(t *testing.T) {
		parent := contextual.New(context.Background())
		defer parent.Cancel()

		ctx, cancel := contextual.WithTimeout(parent, 1*time.Second)

		cancel() // Call cancel immediately

		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("WithTimeout() error after cancel = %v, want %v", ctx.Err(), context.Canceled)
		}
		if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, context.Canceled) {
			t.Errorf("WithTimeout() cause after cancel = %v, want %v", cause, context.Canceled)
		}
	})

	t.Run("parent_cancel_called", func(t *testing.T) {
		parent := contextual.New(context.Background())

		ctx, cancel := contextual.WithTimeout(parent, 1*time.Second)
		defer cancel()

		parent.Cancel() // Cancel parent

		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.Canceled) { // Should be Canceled as parent was canceled
			t.Errorf("WithTimeout() error after parent cancel = %v, want %v", ctx.Err(), context.Canceled)
		}
		if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, context.Canceled) {
			t.Errorf("WithTimeout() cause after parent cancel = %v, want %v", cause, context.Canceled)
		}
	})
}

func TestContextWithTimeoutCause(t *testing.T) {
	testError := errors.New("test error for timeout")

	t.Run("timeout_exceeded_with_cause", func(t *testing.T) {
		parent := contextual.New(context.Background())
		defer parent.Cancel()

		ctx, cancel := contextual.WithTimeoutCause(parent, 1*time.Millisecond, testError)
		defer cancel()

		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) { // Err is still DeadlineExceeded
			t.Errorf("WithTimeoutCause() error = %v, want %v", ctx.Err(), context.DeadlineExceeded)
		}
		if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, testError) {
			t.Errorf("WithTimeoutCause() cause = %v, want %v", cause, testError)
		}
	})

	t.Run("cancel_func_called", func(t *testing.T) {
		parent := contextual.New(context.Background())
		defer parent.Cancel()

		ctx, cancel := contextual.WithTimeoutCause(parent, 1*time.Second, testError)

		cancel() // Call cancel immediately, cause should be Canceled, not testError

		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("WithTimeoutCause() error after cancel = %v, want %v", ctx.Err(), context.Canceled)
		}
		if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, context.Canceled) {
			t.Errorf("WithTimeoutCause() cause after cancel = %v, want %v", cause, context.Canceled)
		}
	})
}

func TestContextWithDeadline(t *testing.T) {
	// This will be similar to WithTimeout but using a specific time
	t.Run("deadline_exceeded", func(t *testing.T) {
		parent := contextual.New(context.Background())
		defer parent.Cancel()

		ctx, cancel := contextual.WithDeadline(parent, time.Now().Add(1*time.Millisecond))
		defer cancel()

		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Errorf("WithDeadline() error = %v, want %v", ctx.Err(), context.DeadlineExceeded)
		}
	})
}


func TestContextWithCancel(t *testing.T) {
	parent := contextual.New(context.Background())
	defer parent.Cancel()

	ctx, cancel := contextual.WithCancel(parent)

	if ctx.Err() != nil {
		t.Fatalf("WithCancel() context has immediate error: %v", ctx.Err())
	}

	cancel() // Call the cancel func

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("WithCancel() error after cancel = %v, want %v", ctx.Err(), context.Canceled)
		}
		if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, context.Canceled) {
			t.Errorf("WithCancel() cause after cancel = %v, want %v", cause, context.Canceled)
		}
	case <-time.After(1 * time.Second):
		t.Error("WithCancel() context did not cancel after cancel() call")
	}

	// Test that parent cancellation also cancels child
	parent2 := contextual.New(context.Background())
	ctx2, cancel2 := contextual.WithCancel(parent2)
	defer cancel2()

	parent2.Cancel()
	select {
	case <-ctx2.Done():
		if !errors.Is(ctx2.Err(), context.Canceled) {
			t.Errorf("WithCancel() error after parent cancel = %v, want %v", ctx2.Err(), context.Canceled)
		}
	case <-time.After(1 * time.Second):
		t.Error("WithCancel() context did not cancel after parent was canceled")
	}
}

func TestContextualFunctionWithCancelCause(t *testing.T) {
	// Test for contextual.WithCancelCause (the function)
	parent := contextual.New(context.Background())
	defer parent.Cancel()

	testErr := errors.New("custom cancel cause")
	ctx, cancelCauseFunc := contextual.WithCancelCause(parent)

	if ctx.Err() != nil {
		t.Fatalf("WithCancelCause() context has immediate error: %v", ctx.Err())
	}

	cancelCauseFunc(testErr) // Call the cancel func with a cause

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) { // Err() still returns Canceled
			t.Errorf("WithCancelCause() error after cancel = %v, want %v", ctx.Err(), context.Canceled)
		}
		if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, testErr) {
			t.Errorf("WithCancelCause() cause after cancel = %v, want %v", cause, testErr)
		}
	case <-time.After(1 * time.Second):
		t.Error("WithCancelCause() context did not cancel after cancelCauseFunc() call")
	}

	// Test with nil cause
	parent3 := contextual.New(context.Background())
	defer parent3.Cancel()
	ctx3, cancel3 := contextual.WithCancelCause(parent3)
	cancel3(nil) // Cancel with nil error
	<-ctx3.Done()
	if cause := context.Cause(ctx3.AsContext()); !errors.Is(cause, context.Canceled) {
		t.Errorf("WithCancelCause() with nil error, cause = %v, want %v", cause, context.Canceled)
	}
}

func TestContextWithSignalCancel(t *testing.T) {
	// Direct signal testing is hard. We test cancellation via stop func and parent.
	t.Run("stop_function_cancels", func(t *testing.T) {
		parent := contextual.New(context.Background())
		defer parent.Cancel()

		// Pass a specific signal to avoid relying on default SIGINT/SIGTERM which might interfere with test runners
		// However, the actual signal doesn't matter if we are calling stop()
		// Calling without specific signals to use the default ones.
		ctx, stop := contextual.WithSignalCancel(parent)

		if ctx.Err() != nil {
			t.Fatalf("WithSignalCancel() context has immediate error: %v", ctx.Err())
		}

		stop() // Call the stop function

		select {
		case <-ctx.Done():
			// When stop is called, the context is canceled.
			// The cause of cancellation by signal.NotifyContext's cancel function is context.Canceled.
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Errorf("WithSignalCancel() error after stop = %v, want %v", ctx.Err(), context.Canceled)
			}
			if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, context.Canceled) {
				 t.Errorf("WithSignalCancel() cause after stop = %v, want %v", cause, context.Canceled)
			}
		case <-time.After(1 * time.Second):
			t.Error("WithSignalCancel() context did not cancel after stop() call")
		}
	})

	t.Run("parent_cancel_cancels_signal_context", func(t *testing.T) {
		parent := contextual.New(context.Background())
		// No defer parent.Cancel() here, we do it manually

		// Calling without specific signals to use the default ones.
		ctx, stop := contextual.WithSignalCancel(parent)
		defer stop() // Ensure stop is called to clean up signal listener

		if ctx.Err() != nil {
			t.Fatalf("WithSignalCancel() context has immediate error: %v", ctx.Err())
		}

		parent.Cancel() // Cancel the parent context

		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Errorf("WithSignalCancel() error after parent cancel = %v, want %v", ctx.Err(), context.Canceled)
			}
		case <-time.After(1 * time.Second):
			t.Error("WithSignalCancel() context did not cancel after parent was canceled")
		}
	})
}
