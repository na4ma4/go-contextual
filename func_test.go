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
