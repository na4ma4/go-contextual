package contextual_test

import (
	"context"
	"errors" // Added import for errors
	"fmt"    // Added import for fmt.Sprintf
	"testing"
	"time"

	"github.com/na4ma4/go-contextual"
)

func TestBackground(t *testing.T) {
	ctx := contextual.Background()
	defer ctx.Cancel()

	go func() {
		time.Sleep(time.Millisecond)
		ctx.Cancel()
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	select {
	case <-ticker.C:
		t.Error("context should have cancelled first")
	case <-ctx.Done():
		t.Log("success context cancelled before ticker")
	}
}

func TestContextCancelWithCauseMethod(t *testing.T) {
	ctx := contextual.New(context.Background())
	// No defer ctx.Cancel() here, we are testing specific cancel with cause

	testErr := errors.New("specific cancellation cause")
	ctx.CancelWithCause(testErr)

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) { // Err() should still report Canceled
			t.Errorf("ctx.Err() after CancelWithCause = %v, want %v", ctx.Err(), context.Canceled)
		}
		retrievedCause := context.Cause(ctx.AsContext())
		if !errors.Is(retrievedCause, testErr) {
			t.Errorf("context.Cause(ctx) = %v, want %v", retrievedCause, testErr)
		}
	case <-time.After(1 * time.Second):
		t.Error("Context did not cancel after CancelWithCause()")
	}

	// Test that cause is not overwritten
	ctx2 := contextual.New(context.Background())
	initialErrorInstance := errors.New("initial cause")
	subsequentErrorInstance := errors.New("subsequent cause attempt")

	ctx2.CancelWithCause(initialErrorInstance)
	select {
	case <-ctx2.Done(): // ensure it's done
	case <-time.After(1 * time.Second):
		t.Fatal("ctx2 did not cancel after initial CancelWithCause")
	}

	ctx2.CancelWithCause(subsequentErrorInstance) // Attempt to overwrite cause
	retrievedCause2 := context.Cause(ctx2.AsContext())

	// Check that the cause is still the initialErrorInstance
	if !errors.Is(retrievedCause2, initialErrorInstance) {
		t.Errorf("Cause after second CancelWithCause = [%v], want [%v]. Initial cause was not preserved.",
			retrievedCause2, initialErrorInstance)
	}
	// Also ensure it's not the subsequentErrorInstance
	if errors.Is(retrievedCause2, subsequentErrorInstance) {
		t.Errorf("Cause after second CancelWithCause = [%v]. Initial cause was overwritten by subsequent error.",
			retrievedCause2)
	}
	// No need for explicit ctx2.Cancel() as it's already cancelled.
}

func TestContextCloneWithNewContext(t *testing.T) {
	// Setup parent context with a value
	type cloneKey string
	const ck cloneKey = "cloneTestKey"
	parentValue := "parentValue"

	originalCtx := contextual.New(context.Background())
	defer originalCtx.Cancel()

	if valStore, ok := originalCtx.(contextual.ContextValueStore); ok {
		valStore.AddValue(ck, parentValue)
	} else {
		t.Fatal("Original context does not implement ContextValueStore")
	}

	// Create a new standard context to be the base for the clone
	// Use WithCancelCause to get a CancelCauseFunc directly
	newStdCtx, newStdCancelCause := context.WithCancelCause(context.Background())
	defer newStdCancelCause(errors.New("deferred cancel for newStdCtx")) // Use an error for cause

	// Clone the context
	clonedCtx := originalCtx.CloneWithNewContext(newStdCtx, newStdCancelCause)
	// Note: The cancel func returned by CloneWithNewContext is actually originalCtx.Cancel,
	// but we are testing the newStdCancel's effect on clonedCtx.

	// 1. Test value inheritance (sharing, due to current implementation)
	if valStore, ok := clonedCtx.(contextual.ContextValueStore); ok {
		retrievedVal, found := valStore.GetE(ck)
		if !found {
			t.Errorf("Cloned context did not find key %q from original", ck)
		}
		if retrievedVal != parentValue {
			t.Errorf("Cloned context GetE(%q) = %v, want %v", ck, retrievedVal, parentValue)
		}
	} else {
		t.Fatal("Cloned context does not implement ContextValueStore")
	}

	// 2. Test cancellation of the cloned context using its new cancel func (newStdCancel)
	if clonedCtx.Err() != nil {
		t.Fatalf("Cloned context has immediate error: %v", clonedCtx.Err())
	}
	newStdCancelCause(errors.New("cloned context specific cancel")) // Cancel the new standard context part
	select {
	case <-clonedCtx.Done():
		if !errors.Is(clonedCtx.Err(), context.Canceled) {
			t.Errorf("Cloned context error after its cancel = %v, want %v", clonedCtx.Err(), context.Canceled)
		}
	case <-time.After(1 * time.Second):
		t.Error("Cloned context did not cancel after its specific cancel func was called")
	}

	// 3. Test that original context is not affected by clonedCtx's cancellation via newStdCancel
	if originalCtx.Err() != nil {
		t.Errorf("Original context was affected by cloned context's specific cancellation: %v", originalCtx.Err())
	}

	// 4. Test cancellation of the original parent context also cancels the cloned context
	// Re-clone for a fresh cancellable instance, as newStdCtx is already cancelled.
	originalCtx2 := contextual.New(context.Background())
	defer originalCtx2.Cancel()

	newStdCtx2, newStdCancelCause2 := context.WithCancelCause(context.Background())
	defer newStdCancelCause2(errors.New("deferred cancel for newStdCtx2"))
	clonedCtx2 := originalCtx2.CloneWithNewContext(newStdCtx2, newStdCancelCause2)

	originalCtx2.Cancel() // Cancel the original contextual parent
	select {
	case <-clonedCtx2.Done():
		// The cloned context's Done channel should be closed because its *effective* parent
		// (originalCtx2, from which errgroup and other properties might be shared or conceptually derived)
		// was cancelled. However, CloneWithNewContext replaces the underlying standard context.
		// The `clonedCtx`'s `Done()` channel is `newStdCtx2.Done()`.
		// `originalCtx2.Cancel()` does not directly affect `newStdCtx2`.
		// This part of the test reveals a nuance: `CloneWithNewContext` truly detaches
		// the cancellation of the new context from the *original* `contextual.Context`'s direct cancel,
		// linking it only to the `newStdCtx` and its cancel func.
		// This is consistent with `context.WithValue` or `context.WithCancel` like behavior.
		// The original design of CloneWithNewContext might need clarification if tighter coupling was expected.
		// For now, we test that originalCtx2.Cancel() does NOT cancel clonedCtx2 directly
		// if newStdCtx2 is independent.
		// *However*, if the `CloneWithNewContext` implementation makes `clonedCtx` a child of `originalCtx.AsContext()`
		// then it *would* be cancelled. The current `CloneWithNewContext` takes `ctx context.Context` and uses it directly.
		// Let's assume `newStdCtx` is NOT a child of `originalCtx2.AsContext()` for this test.
		// So, clonedCtx2 should NOT be done here.
		if clonedCtx2.Err() != nil {
             t.Errorf("Cloned context (clonedCtx2) was unexpectedly cancelled by originalCtx2.Cancel(): %v", clonedCtx2.Err())
        }
	case <-time.After(50 * time.Millisecond):
		// This is the expected path if newStdCtx2 is independent of originalCtx2
		t.Log("Cloned context (clonedCtx2) correctly not cancelled by originalCtx2.Cancel(), as its underlying context is newStdCtx2.")
	}

	// 5. Test if the cancel func returned by originalCtx.CloneWithNewContext can cancel the clone.
	// The current implementation of CloneWithNewContext returns a Context that, when its own .Cancel()
	// or .CancelWithCause() is called, it uses the `cancel context.CancelCauseFunc` provided
	// during its creation (i.e., newStdCancel in this test).
	// The `originalCtx.Cancel` is for `originalCtx`.
	// The `clonedCtx.Cancel` should trigger `newStdCancel`.

	newStdCtx3, newStdCancelCause3 := context.WithCancelCause(context.Background())
	// No defer newStdCancelCause3, we'll use clonedCtx3.Cancel()

	originalCtx3 := contextual.New(context.Background())
	defer originalCtx3.Cancel()

	clonedCtx3 := originalCtx3.CloneWithNewContext(newStdCtx3, newStdCancelCause3)
	clonedCtx3.CancelWithCause(errors.New("clonedCtx3 cancel method")) // This should call newStdCancelCause3

	select {
	case <-clonedCtx3.Done():
		if !errors.Is(clonedCtx3.Err(), context.Canceled) {
			t.Errorf("Cloned context (clonedCtx3) error after its Cancel() = %v, want %v", clonedCtx3.Err(), context.Canceled)
		}
	case <-time.After(1 * time.Second):
		t.Error("Cloned context (clonedCtx3) did not cancel after its own Cancel() method was called")
	}
}

func TestContextConditionalRunner(t *testing.T) {
	ctx := contextual.New(context.Background())
	defer ctx.Cancel()

	runner, ok := ctx.(contextual.ContextConditionalRunner)
	if !ok {
		t.Fatal("Context does not implement ContextConditionalRunner")
	}

	type condKey contextual.ContextKeyBool
	const (
		runKeyTrue  condKey = "runKeyTrue"
		runKeyFalse condKey = "runKeyFalse"
		runKeyNotSet condKey = "runKeyNotSet"
	)

	var trueExecuted, falseExecuted, notSetExecuted bool

	// Set keys
	runner.SetContextKey(contextual.ContextKeyBool(runKeyTrue), true)
	runner.SetContextKey(contextual.ContextKeyBool(runKeyFalse), false)

	// Test RunIf
	runner.RunIf(contextual.ContextKeyBool(runKeyTrue), func() {
		trueExecuted = true
	})
	runner.RunIf(contextual.ContextKeyBool(runKeyFalse), func() {
		falseExecuted = true
	})
	runner.RunIf(contextual.ContextKeyBool(runKeyNotSet), func() {
		notSetExecuted = true
	})

	if !trueExecuted {
		t.Error("RunIf: function for true key was not executed")
	}
	if falseExecuted {
		t.Error("RunIf: function for false key was executed")
	}
	if notSetExecuted {
		t.Error("RunIf: function for non-set key was executed")
	}

	// Test that SetContextKey actually uses the underlying value store if possible
	// by trying to retrieve it via ContextValueStore (this is an implementation detail check)
	valStore, ok := ctx.(contextual.ContextValueStore)
	if !ok {
		t.Log("Context does not implement ContextValueStore, skipping check on SetContextKey's underlying storage")
	} else {
		// Ensure the key used for retrieval has the exact same type as used in SetContextKey
		val, found := valStore.GetE(contextual.ContextKeyBool(runKeyTrue))
		if !found {
			t.Errorf("SetContextKey(%q, true) did not store value", runKeyTrue)
		}
		boolVal, isBool := val.(bool)
		if !isBool || !boolVal {
			t.Errorf("SetContextKey(%q, true) stored %v (%T), want true (bool)", runKeyTrue, val, val)
		}
	}
}

func TestContextValueStore(t *testing.T) {
	ctx := contextual.New(context.Background())
	defer ctx.Cancel()

	type storeKey string
	const (
		keyString storeKey = "myString"
		keyInt    storeKey = "myInt"
		keyStruct storeKey = "myStruct"
		keyNonEx  storeKey = "myNonExistentKey"
	)

	myStringVal := "hello world"
	myIntVal := 123
	myStructVal := struct{ Name string }{Name: "Test"}

	valStore, ok := ctx.(contextual.ContextValueStore)
	if !ok {
		t.Fatal("Context does not implement ContextValueStore")
	}

	// AddValue
	valStore.AddValue(keyString, myStringVal)
	valStore.AddValue(keyInt, myIntVal)
	valStore.AddValue(keyStruct, myStructVal)

	// GetE - existing
	retStringE, okStringE := valStore.GetE(keyString)
	if !okStringE {
		t.Errorf("GetE(%q) ok = false, want true", keyString)
	}
	if retStringE != myStringVal {
		t.Errorf("GetE(%q) val = %v, want %v", keyString, retStringE, myStringVal)
	}

	// GetE - non-existing
	_, okNonExE := valStore.GetE(keyNonEx)
	if okNonExE {
		t.Errorf("GetE(%q) ok = true, want false", keyNonEx)
	}

	// Get - existing
	retInt := valStore.Get(keyInt)
	if retInt != myIntVal {
		t.Errorf("Get(%q) val = %v, want %v", keyInt, retInt, myIntVal)
	}

	// Get - non-existing
	retNonEx := valStore.Get(keyNonEx)
	if retNonEx != nil {
		t.Errorf("Get(%q) val = %v, want nil", keyNonEx, retNonEx)
	}

	// GetString - existing string
	s := valStore.GetString(keyString)
	if s != myStringVal {
		t.Errorf("GetString(%q) = %q, want %q", keyString, s, myStringVal)
	}

	// GetString - existing int (should format)
	sInt := valStore.GetString(keyInt)
	expectedSInt := fmt.Sprintf("%v", myIntVal)
	if sInt != expectedSInt {
		t.Errorf("GetString(%q) for int = %q, want %q", keyInt, sInt, expectedSInt)
	}

	// GetString - non-existing
	sNonEx := valStore.GetString(keyNonEx)
	if sNonEx != "" {
		t.Errorf("GetString(%q) = %q, want \"\"", keyNonEx, sNonEx)
	}

	// GetInt - existing int
	i := valStore.GetInt(keyInt)
	if i != myIntVal {
		t.Errorf("GetInt(%q) = %d, want %d", keyInt, i, myIntVal)
	}

	// GetInt - existing string (should parse)
	valStore.AddValue("intString", "456")
	iStr := valStore.GetInt("intString")
	if iStr != 456 {
		t.Errorf("GetInt(%q) for string \"456\" = %d, want 456", "intString", iStr)
	}

	valStore.AddValue("invalidIntString", "not-an-int")
	iInvalidStr := valStore.GetInt("invalidIntString")
	if iInvalidStr != 0 {
		t.Errorf("GetInt(%q) for string \"not-an-int\" = %d, want 0", "invalidIntString", iInvalidStr)
	}

	// GetInt - non-existing
	iNonEx := valStore.GetInt(keyNonEx)
	if iNonEx != 0 {
		t.Errorf("GetInt(%q) = %d, want 0", keyNonEx, iNonEx)
	}

	// GetInt - struct (should be 0)
	iStruct := valStore.GetInt(keyStruct)
	if iStruct != 0 {
		t.Errorf("GetInt(%q) for struct = %d, want 0", keyStruct, iStruct)
	}

	// Overwrite value
	valStore.AddValue(keyString, "new string")
	newS := valStore.GetString(keyString)
	if newS != "new string" {
		t.Errorf("GetString after overwrite = %q, want \"new string\"", newS)
	}
}

func TestAllowNilNewCancellable(t *testing.T) {
	ctx := contextual.NewCancellable(nil) //nolint:staticcheck // testing bad juju.
	if ctx == nil {
		t.Fatal("context should return valid text")
	}
	defer ctx.Cancel()

	go func() {
		time.Sleep(time.Millisecond)
		ctx.Cancel()
	}()

	timeout := time.NewTicker(time.Second)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		t.Error("context should have cancelled first")
	case <-ctx.Done():
		t.Log("success context cancelled before timeout")
	}
}

func TestReplaceContext(t *testing.T) {
	ctx := contextual.Background()
	defer ctx.Cancel()

	newCtx, cancel := context.WithCancel(context.Background())
	ctx.ReplaceContext(func(_ context.Context) context.Context {
		return newCtx
	})
	defer cancel()

	timeout := time.NewTicker(time.Second)
	defer timeout.Stop()

	cancel()

	select {
	case <-timeout.C:
		t.Error("context should have cancelled first")
	case <-ctx.Done():
		t.Log("success context cancelled before timeout")
	}
}
