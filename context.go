// Copyright (c) 2026 blairtcg
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package velo

import "context"

type contextKey struct{ string }

// _contextkeyinstance is the key used to store the logger in context.
var _contextKeyInstance = contextKey{"log"}

// WithContext injects the provided Logger into the given context.
//
// It returns a new context containing the Logger. Use this to pass a
// specifically configured Logger (e.g., one with request scoped fields) down
// the call stack without modifying function signatures.
func WithContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, _contextKeyInstance, logger)
}

// FromContext extracts the Logger from the provided context.
//
// It returns the global default Logger if the context does not contain one.
// Use this to retrieve a request scoped Logger injected by middleware. This
// ensures your application always has a valid Logger instance to write to.
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(_contextKeyInstance).(*Logger); ok {
		return logger
	}
	return Default()
}
