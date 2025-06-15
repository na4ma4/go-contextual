package contextual_test

import (
	"context"
	"errors"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/na4ma4/go-contextual"
)

var errTest = errors.New("test error")

func TestGo_FuncErr(t *testing.T) {
	ctx := contextual.New(t.Context())
	defer ctx.Cancel()

	var executed bool
	contextual.Go(ctx, contextual.FuncErr(func() error {
		executed = true
		return nil
	}))

	if err := ctx.Wait(); err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
	if !executed {
		t.Error("FuncErr was not executed")
	}

	ctxError := contextual.New(t.Context())
	defer ctxError.Cancel()
	contextual.Go(ctxError, contextual.FuncErr(func() error {
		return errTest
	}))

	if err := ctxError.Wait(); !errors.Is(err, errTest) {
		t.Errorf("Wait() error = %v, want %v", err, errTest)
	}
}

func TestGo_CtxErrFunc(t *testing.T) {
	ctx := contextual.New(t.Context())
	defer ctx.Cancel()

	var executed bool
	var receivedCtx context.Context
	contextual.Go(ctx, contextual.CtxErrFunc(func(c context.Context) error {
		executed = true
		//nolint:fatcontext // Testing context being received.
		receivedCtx = c
		return nil
	}))

	if err := ctx.Wait(); err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
	if !executed {
		t.Error("CtxErrFunc was not executed")
	}
	if receivedCtx == nil {
		t.Error("CtxErrFunc did not receive context")
	}

	ctxError := contextual.New(t.Context())
	defer ctxError.Cancel()
	contextual.Go(ctxError, contextual.CtxErrFunc(func(_ context.Context) error {
		return errTest
	}))

	if err := ctxError.Wait(); !errors.Is(err, errTest) {
		t.Errorf("Wait() error = %v, want %v", err, errTest)
	}
}

func TestGo_CtxualErrFunc(t *testing.T) {
	ctx := contextual.New(t.Context())
	defer ctx.Cancel()

	var executed bool
	var receivedCtx contextual.Context
	contextual.Go(ctx, contextual.CtxualErrFunc(func(c contextual.Context) error {
		executed = true
		receivedCtx = c
		c.Go(func() error { // This is ctx.Go(), not generic contextual.Go()
			return nil
		})
		return nil
	}))

	if err := ctx.Wait(); err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
	if !executed {
		t.Error("CtxualErrFunc was not executed")
	}
	if receivedCtx == nil {
		t.Error("CtxualErrFunc did not receive contextual.Context")
	}

	ctxError := contextual.New(t.Context())
	defer ctxError.Cancel()
	contextual.Go(ctxError, contextual.CtxualErrFunc(func(_ contextual.Context) error {
		return errTest
	}))

	if err := ctxError.Wait(); !errors.Is(err, errTest) {
		t.Errorf("Wait() error = %v, want %v", err, errTest)
	}
}

func TestGo_Cancellation(t *testing.T) {
	ctx := contextual.New(t.Context())

	var wg sync.WaitGroup
	wg.Add(1)

	started := make(chan struct{})
	contextual.Go(ctx, contextual.CtxErrFunc(func(c context.Context) error {
		close(started)
		wg.Done()
		select {
		case <-c.Done():
			return c.Err()
		case <-time.After(5 * time.Second):
			return errors.New("goroutine did not cancel in time")
		}
	}))

	<-started
	ctx.Cancel()

	err := ctx.Wait()
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Wait() error = %v, want %v", err, context.Canceled)
	}
	wg.Wait()
}

func TestGoLabelled(t *testing.T) {
	ctx := contextual.New(t.Context())
	defer ctx.Cancel()

	var executed bool
	contextual.GoLabelled(ctx, "testjob", "testdesc", contextual.FuncErr(func() error {
		executed = true
		return nil
	}))

	if err := ctx.Wait(); err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
	if !executed {
		t.Error("GoLabelled FuncErr was not executed")
	}

	ctxError := contextual.New(t.Context())
	defer ctxError.Cancel()
	contextual.GoLabelled(ctxError, "testjobErr", "testdescErr", contextual.FuncErr(func() error {
		return errTest
	}))

	if err := ctxError.Wait(); !errors.Is(err, errTest) {
		t.Errorf("Wait() error = %v, want %v", err, errTest)
	}

	ctxCtxual := contextual.New(t.Context())
	defer ctxCtxual.Cancel()
	var executedCtxual bool
	contextual.GoLabelled(
		ctxCtxual, "testjobCtxual", "testdescCtxual",
		contextual.CtxualErrFunc(
			func(c contextual.Context) error {
				if c == nil {
					return errors.New("context in GoLabelled was nil")
				}
				executedCtxual = true
				return nil
			},
		),
	)
	if err := ctxCtxual.Wait(); err != nil {
		t.Errorf("Wait() error for CtxualErrFunc in GoLabelled = %v, want nil", err)
	}
	if !executedCtxual {
		t.Error("GoLabelled CtxualErrFunc was not executed")
	}
}

func TestGo_MultipleGoroutines(t *testing.T) {
	ctx := contextual.New(t.Context())
	defer ctx.Cancel()

	numGoroutines := 5
	var executedCount atomic.Int32 // Using int32 for atomic, though not using atomic here for tool simplicity
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		t.Logf("Starting goroutine %d", i)
		contextual.Go(ctx, contextual.FuncErr(func() error {
			// In a real test, use atomic.AddInt32(&executedCount, 1)
			func() { // Simulating work and avoiding data race for executedCount for this test
				executedCount.Add(1)
				wg.Done()
			}()
			time.Sleep(10 * time.Millisecond)
			return nil
		}))
	}

	err := ctx.Wait()
	if err != nil {
		t.Fatalf("Wait() returned an error: %v", err)
	}
	wg.Wait() // Ensure all goroutines have at least run their increment logic

	if executedCount.Load() != int32(numGoroutines) {
		t.Errorf("Expected %d goroutines to execute, but got %d", numGoroutines, executedCount.Load())
	}
}

func TestGo_ErrorCancelsOthers(t *testing.T) {
	ctx := contextual.New(t.Context())

	errEarly := errors.New("early error")
	var slowTaskStarted bool
	var slowTaskCancelled bool

	contextual.Go(ctx, contextual.FuncErr(func() error {
		return errEarly
	}))

	contextual.Go(ctx, contextual.CtxErrFunc(func(c context.Context) error {
		slowTaskStarted = true
		select {
		case <-time.After(2 * time.Second):
			return errors.New("slow task was not cancelled")
		case <-c.Done():
			slowTaskCancelled = true
			return context.Cause(c)
		}
	}))

	err := ctx.Wait()
	if !errors.Is(err, errEarly) {
		t.Errorf("Wait() error = %v, want %v", err, errEarly)
	}

	if !slowTaskStarted {
		t.Error("Slow task did not even start")
	}
	if !slowTaskCancelled {
		t.Error("Slow task was not cancelled by the early error")
	}
}

func TestGo_CtxualErrFunc_WithValueAccess(t *testing.T) {
	type key string
	const myKey key = "testKey"
	ctx := contextual.New(t.Context(), contextual.WithValues([]contextual.ContextKV{{Key: myKey, Value: "testValue"}}))
	defer ctx.Cancel()

	var foundValue string

	contextual.Go(ctx, contextual.CtxualErrFunc(func(c contextual.Context) error {
		if vs, ok := c.(contextual.ContextValueStore); ok {
			foundValue = vs.GetString(myKey)
		} else {
			return errors.New("context does not implement ContextValueStore")
		}
		return nil
	}))

	if err := ctx.Wait(); err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}

	if foundValue != "testValue" {
		t.Errorf("GetString(myKey) = %s, want 'testValue'", foundValue)
	}
}

func TestGo_NilFunc(t *testing.T) {
	ctx := contextual.New(t.Context())
	defer ctx.Cancel()

	t.Run("Go_FuncErr_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("contextual.Go(ctx, (FuncErr)(nil)) did not panic")
			}
		}()
		contextual.Go(ctx, (contextual.FuncErr)(nil))
		ctx.Wait()
	})

	ctx = contextual.New(t.Context()) // Re-init ctx for isolation
	defer ctx.Cancel()
	t.Run("Go_CtxErrFunc_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("contextual.Go(ctx, (CtxErrFunc)(nil)) did not panic")
			}
		}()
		contextual.Go(ctx, (contextual.CtxErrFunc)(nil))
		ctx.Wait()
	})

	ctx = contextual.New(t.Context()) // Re-init ctx
	defer ctx.Cancel()
	t.Run("Go_CtxualErrFunc_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("contextual.Go(ctx, (CtxualErrFunc)(nil)) did not panic")
			}
		}()
		contextual.Go(ctx, (contextual.CtxualErrFunc)(nil))
		ctx.Wait()
	})

	ctx = contextual.New(t.Context()) // Re-init ctx
	defer ctx.Cancel()
	t.Run("GoLabelled_FuncErr_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("contextual.GoLabelled(ctx, ..., (FuncErr)(nil)) did not panic")
			}
		}()
		contextual.GoLabelled(ctx, "n", "d", (contextual.FuncErr)(nil))
		ctx.Wait()
	})
}

func TestGo_CtxErrFunc_ContextPropagation(t *testing.T) {
	type key string
	const baseKey key = "baseKey"
	base := context.WithValue(t.Context(), baseKey, "baseValue")
	ctx := contextual.New(base)
	defer ctx.Cancel()

	var receivedCtxVal any
	var wg sync.WaitGroup
	wg.Add(1)

	contextual.Go(ctx, contextual.CtxErrFunc(func(c context.Context) error {
		defer wg.Done()
		receivedCtxVal = c.Value(baseKey)
		if receivedCtxVal == nil {
			return errors.New("Value(baseKey) was nil in CtxErrFunc")
		}
		select {
		case <-c.Done():
			return context.Cause(c)
		case <-time.After(1 * time.Second):
		}
		return nil
	}))

	time.AfterFunc(50*time.Millisecond, func() {
		ctx.CancelWithCause(errTest)
	})

	waitErr := ctx.Wait()
	wg.Wait()

	if !errors.Is(waitErr, errTest) {
		t.Errorf("ctx.Wait() err = %v; want %v", waitErr, errTest)
	}
	if receivedCtxVal != "baseValue" {
		t.Errorf("Received context value = %v, want 'baseValue'", receivedCtxVal)
	}
}

func TestGo_CtxualErrFunc_ContextPropagation(t *testing.T) {
	type key string
	const myKey key = "myKey"
	const myValue string = "myValue"

	ctx := contextual.New(t.Context(), contextual.WithValues([]contextual.ContextKV{{Key: myKey, Value: myValue}}))
	defer ctx.Cancel()

	var foundVal string
	var wg sync.WaitGroup
	wg.Add(1)

	contextual.Go(ctx, contextual.CtxualErrFunc(func(c contextual.Context) error {
		defer wg.Done()
		if vs, ok := c.(contextual.ContextValueStore); ok {
			foundVal = vs.GetString(myKey)
		} else {
			return errors.New("context is not ContextValueStore")
		}
		select {
		case <-c.Done():
			return context.Cause(c)
		case <-time.After(1 * time.Second):
		}
		return nil
	}))

	time.AfterFunc(50*time.Millisecond, func() {
		ctx.CancelWithCause(errTest)
	})

	waitErr := ctx.Wait()
	wg.Wait()

	if !errors.Is(waitErr, errTest) {
		t.Errorf("ctx.Wait() err = %v; want %v", waitErr, errTest)
	}
	if foundVal != myValue {
		t.Errorf("GetString(myKey) = %q, want %q", foundVal, myValue)
	}
}

func TestGoLabelled_CtxErrFunc_ContextPropagation(t *testing.T) {
	type key string
	const baseKey key = "baseKey"
	base := context.WithValue(t.Context(), baseKey, "baseValue")
	ctx := contextual.New(base)
	defer ctx.Cancel()

	var receivedCtxVal any
	var wg sync.WaitGroup
	wg.Add(1)

	contextual.GoLabelled(ctx, "job", "desc", contextual.CtxErrFunc(func(c context.Context) error {
		defer wg.Done()
		receivedCtxVal = c.Value(baseKey)
		if receivedCtxVal == nil {
			return errors.New("Value(baseKey) was nil in CtxErrFunc (labelled)")
		}
		select {
		case <-c.Done():
			return context.Cause(c)
		case <-time.After(1 * time.Second):
		}
		return nil
	}))

	time.AfterFunc(50*time.Millisecond, func() {
		ctx.CancelWithCause(errTest)
	})

	waitErr := ctx.Wait()
	wg.Wait()

	if !errors.Is(waitErr, errTest) {
		t.Errorf("ctx.Wait() err = %v; want %v", waitErr, errTest)
	}
	if receivedCtxVal != "baseValue" {
		t.Errorf("Received context value = %v, want 'baseValue'", receivedCtxVal)
	}
}

func TestGoLabelled_CtxualErrFunc_ContextPropagation(t *testing.T) {
	type key string
	const myKey key = "myKey"
	const myValue string = "myValue"

	ctx := contextual.New(t.Context(), contextual.WithValues([]contextual.ContextKV{{Key: myKey, Value: myValue}}))
	defer ctx.Cancel()

	var foundVal string
	var wg sync.WaitGroup
	wg.Add(1)

	contextual.GoLabelled(ctx, "job", "desc", contextual.CtxualErrFunc(func(c contextual.Context) error {
		defer wg.Done()
		if vs, ok := c.(contextual.ContextValueStore); ok {
			foundVal = vs.GetString(myKey)
		} else {
			return errors.New("context is not ContextValueStore (labelled)")
		}
		select {
		case <-c.Done():
			return context.Cause(c)
		case <-time.After(1 * time.Second): // Corrected: time.After
		}
		return nil
	}))

	time.AfterFunc(50*time.Millisecond, func() {
		ctx.CancelWithCause(errTest)
	})

	waitErr := ctx.Wait()
	wg.Wait()

	if !errors.Is(waitErr, errTest) {
		t.Errorf("ctx.Wait() err = %v; want %v", waitErr, errTest)
	}
	if foundVal != myValue {
		t.Errorf("GetString(myKey) = %q, want %q", foundVal, myValue)
	}
}

func TestGoDirectlyOnContext(t *testing.T) {
	ctx := contextual.New(t.Context())
	defer ctx.Cancel()

	var executed bool
	ctx.Go(func() error {
		executed = true
		return nil
	})

	if err := ctx.Wait(); err != nil {
		t.Errorf("ctx.Go().Wait() error = %v, want nil", err)
	}
	if !executed {
		t.Error("ctx.Go() was not executed")
	}

	ctxError := contextual.New(t.Context())
	defer ctxError.Cancel()
	ctxError.Go(func() error {
		return errTest
	})

	if err := ctxError.Wait(); !errors.Is(err, errTest) {
		t.Errorf("ctx.Go().Wait() error = %v, want %v", err, errTest)
	}

	ctxLabelled := contextual.New(t.Context())
	defer ctxLabelled.Cancel()

	var executedLabelled bool
	ctxLabelled.GoLabelled(pprof.Labels("test", "direct"), func() error {
		executedLabelled = true
		return nil
	})

	if err := ctxLabelled.Wait(); err != nil {
		t.Errorf("ctx.GoLabelled().Wait() error = %v, want nil", err)
	}
	if !executedLabelled {
		t.Error("ctx.GoLabelled() was not executed")
	}
}
