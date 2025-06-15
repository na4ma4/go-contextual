package contextual_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/na4ma4/go-contextual"
)

func TestContextOptionTimeout(t *testing.T) {
	ctx := contextual.NewCancellable(
		context.TODO(),
		contextual.WithTimeoutOption(50*time.Millisecond),
	)
	if err := ctx.Err(); err != nil {
		t.Errorf("ctx.Err(): immediate error : got '%s', want 'nil'", err)
	}
	time.Sleep(time.Second)
	if err := ctx.Err(); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("ctx.Err(): immediate error : got '%s', want 'Deadline Exceeded'", err)
	}
}

func TestContextOptionDeadline(t *testing.T) {
	deadline := time.Now().Add(50 * time.Millisecond)
	ctx := contextual.New( // Using New for variety, NewCancellable also fine
		context.Background(),
		contextual.WithDeadlineOption(deadline),
	)
	defer ctx.Cancel() // Good practice, though timeout should hit first

	// Check status before deadline
	if err := ctx.Err(); err != nil {
		t.Fatalf("ctx.Err() before deadline: got '%s', want 'nil'", err)
	}

	// Wait until after the deadline
	time.Sleep(100 * time.Millisecond)

	if err := ctx.Err(); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("ctx.Err() after deadline: got '%s', want '%v'", err, context.DeadlineExceeded)
	}
	if cause := context.Cause(ctx.AsContext()); !errors.Is(cause, context.DeadlineExceeded) {
		t.Errorf("Cause after deadline: got '%s', want '%v'", cause, context.DeadlineExceeded)
	}
}

func TestContextOptionWithValues(t *testing.T) {
	type valKey string
	const k1 valKey = "key1"
	const k2 valKey = "key2"

	val1 := "value1"
	val2 := 123

	ctx := contextual.New(
		context.Background(),
		contextual.WithValues([]contextual.ContextKV{
			{Key: k1, Value: val1},
			{Key: k2, Value: val2},
		}),
	)
	defer ctx.Cancel()

	valStore, ok := ctx.(contextual.ContextValueStore)
	if !ok {
		t.Fatal("Context does not implement ContextValueStore")
	}

	retVal1, found1 := valStore.GetE(k1)
	if !found1 || retVal1 != val1 {
		t.Errorf("GetE(%q) = %v, %t, want %v, true", k1, retVal1, found1, val1)
	}

	retVal2 := valStore.GetInt(k2)
	if retVal2 != val2 {
		t.Errorf("GetInt(%q) = %d, want %d", k2, retVal2, val2)
	}

	// Test that standard context.Value does not see these values
	// (as they are in the custom store)
	if v := ctx.Value(k1); v != nil {
		t.Errorf("ctx.Value(%q) got %v, want nil (should be in custom store only)", k1, v)
	}
}

func TestContextOptionWithCustomCancelFunc(t *testing.T) {
	var customCancelCalled bool
	customFunc := func() {
		customCancelCalled = true
	}

	ctx := contextual.New(
		context.Background(),
		contextual.WithCustomCancelFunc(customFunc),
	)

	ctx.Cancel() // Trigger cancellation

	select {
	case <-ctx.Done():
		if !customCancelCalled {
			t.Error("WithCustomCancelFunc: custom function was not called on cancel")
		}
	case <-time.After(1 * time.Second):
		t.Error("WithCustomCancelFunc: context did not cancel")
	}
}

func TestContextOptionWithCustomCancelCauseFunc(t *testing.T) {
	var customCancelCalled bool
	var receivedCause error
	testErr := errors.New("custom-cause-test")

	customFunc := func(cause error) {
		customCancelCalled = true
		receivedCause = cause
	}

	ctx := contextual.New(
		context.Background(),
		contextual.WithCustomCancelCauseFunc(customFunc),
	)

	ctx.CancelWithCause(testErr) // Trigger cancellation with a specific cause

	select {
	case <-ctx.Done():
		if !customCancelCalled {
			t.Error("WithCustomCancelCauseFunc: custom function was not called on cancel")
		}
		if !errors.Is(receivedCause, testErr) {
			t.Errorf("WithCustomCancelCauseFunc: custom function received cause %v, want %v", receivedCause, testErr)
		}
	case <-time.After(1 * time.Second):
		t.Error("WithCustomCancelCauseFunc: context did not cancel")
	}

	// Test with regular Cancel()
	customCancelCalled = false
	receivedCause = nil
	ctx2 := contextual.New(
		context.Background(),
		contextual.WithCustomCancelCauseFunc(customFunc),
	)
	ctx2.Cancel()
	<-ctx2.Done()
	if !customCancelCalled {
		t.Error("WithCustomCancelCauseFunc (on regular Cancel): custom function was not called")
	}
	if !errors.Is(receivedCause, context.Canceled) { // Default cause for .Cancel()
		t.Errorf("WithCustomCancelCauseFunc (on regular Cancel): custom function received cause %v, want %v", receivedCause, context.Canceled)
	}
}

func TestContextOptionWithPProfLabels(t *testing.T) {
	// Basic test: does it create without error?
	// Verifying actual label application is complex for unit tests.
	ctx := contextual.New(
		context.Background(),
		contextual.WithPProfLabels(contextual.Labels("key", "value")),
	)
	defer ctx.Cancel()

	if ctx == nil {
		t.Fatal("WithPProfLabels: context creation returned nil")
	}
	// Ensure it's still cancellable
	go func() { time.Sleep(10 * time.Millisecond); ctx.Cancel() }()
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(1 * time.Second):
		t.Error("WithPProfLabels: context did not cancel")
	}
}

func TestContextOptionWithSignalCancel(t *testing.T) {
	// Similar to TestContextWithSignalCancel, we test cancellation via context's own cancel
	// and parent cancellation, not actual OS signals.
	t.Run("option_self_cancel", func(t *testing.T) {
		ctx := contextual.New(
			context.Background(),
			contextual.WithSignalCancelOption(), // Use default signals
		)

		if ctx.Err() != nil {
			t.Fatalf("WithSignalCancelOption context has immediate error: %v", ctx.Err())
		}

		ctx.Cancel() // Call the context's own cancel function

		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Errorf("WithSignalCancelOption error after self cancel = %v, want %v", ctx.Err(), context.Canceled)
			}
		case <-time.After(1 * time.Second):
			t.Error("WithSignalCancelOption context did not cancel after self cancel call")
		}
	})

	t.Run("option_parent_cancel", func(t *testing.T) {
		parent := contextual.New(context.Background())
		// No defer parent.Cancel() here

		ctx := contextual.New(
			parent, // Use the cancellable parent
			contextual.WithSignalCancelOption(),
		)
		// The WithSignalCancelOption returns a new context whose lifetime might be tied
		// to the one from signal.NotifyContext. The stop function for that is pushed
		// onto the cancel chain of the context *returned by WithSignalCancelOption*.
		// So, parent.Cancel() should propagate.

		parent.Cancel() // Cancel the parent context

		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Errorf("WithSignalCancelOption error after parent cancel = %v, want %v", ctx.Err(), context.Canceled)
			}
		case <-time.After(1 * time.Second):
			t.Error("WithSignalCancelOption context did not cancel after parent was canceled")
		}
	})
}

func TestContextCancel(t *testing.T) {
	ctx := contextual.NewCancellable(context.TODO())
	if err := ctx.Err(); err != nil {
		t.Errorf("ctx.Err(): immediate error : got '%s', want 'nil'", err)
	}
	ctx.Cancel()
	if err := ctx.Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("ctx.Err(): immediate error : got '%s', want 'Canceled'", err)
	}
}
