package velo

import "context"

type contextKey struct{ string }

// _contextKeyInstance is the internal key used to store the logger in a context.
var _contextKeyInstance = contextKey{"log"}

// WithContext adds a logger to a context.
//
// It returns a new context that contains the provided logger. Use this
// function to pass a specific logger down your application's call stack
// without changing your function signatures.
func WithContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, _contextKeyInstance, logger)
}

// FromContext retrieves the logger from a context.
//
// If the context contains a logger, this function returns it. If the
// context does not contain a logger, it returns the default package
// logger instead. This ensures that you always receive a working logger
// when you call it.
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(_contextKeyInstance).(*Logger); ok {
		return logger
	}
	return Default()
}
