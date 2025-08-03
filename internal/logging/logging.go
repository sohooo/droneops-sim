package logging

import (
	"context"
	"log/slog"
	"os"
)

// New returns a logger configured with a text handler writing to STDOUT.
func New() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

type ctxKey struct{}

// NewContext returns a copy of ctx with the logger stored.
func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves a logger from ctx or returns slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
