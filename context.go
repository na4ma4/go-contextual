package contextual

import (
	"context"
)

type Context interface {
	context.Context

	Cancel()
	CancelWithCause(err error)
	CloneWithNewContext(ctx context.Context, cancel context.CancelCauseFunc) Context
	Go(f func() error)
	Wait() error
	// Health() health.Health
	ReplaceContext(cb func(context.Context) context.Context)
}

type ContextValueStore interface {
	AddValue(key any, value any)
	GetE(key any) (any, bool)
	Get(key any) any
	GetString(key any) string
	GetInt(key any) int
}
