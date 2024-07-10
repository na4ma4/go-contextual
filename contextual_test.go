package contextual_test

import (
	"context"
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
