package contextual

import (
	"context"
	"runtime/pprof"
)

type Context interface {
	context.Context

	Cancel()
	CancelWithCause(err error)
	CloneWithNewContext(ctx context.Context, cancel context.CancelCauseFunc) Context
	Go(f func() error)
	GoLabelled(labelSet pprof.LabelSet, f func() error)
	Wait() error
	// Health() health.Health
	ReplaceContext(cb func(context.Context) context.Context)
	AsContext() context.Context
}

type ContextCancelMod interface {
	PushCancelCauseFunc(f context.CancelCauseFunc)
	PushCancelFunc(f context.CancelFunc)
}

type ContextConditionalRunner interface {
	SetContextKey(key ContextKeyBool, value bool)
	RunIf(key ContextKeyBool, f func())
}

type ContextValueStore interface {
	AddValue(key any, value any)
	GetE(key any) (any, bool)
	Get(key any) any
	GetString(key any) string
	GetInt(key any) int
}

func New(ctx context.Context, opts ...OptionFunc) Context {
	return NewCancellable(ctx, opts...)
}
