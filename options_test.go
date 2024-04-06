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
