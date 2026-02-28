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

import (
	"context"
	"log/slog"
)

// SlogHandler adapts a Velo Logger to satisfy the standard library's slog.Handler interface.
//
// This allows you to use Velo as the high performance backend for any code
// that relies on the standard log/slog package.
type SlogHandler struct {
	logger *Logger
	attrs  []Field
	group  string
}

// NewSlogHandler initializes a new SlogHandler using the provided Velo Logger.
func NewSlogHandler(logger *Logger) *SlogHandler {
	return &SlogHandler{logger: logger}
}

// Enabled determines if the handler should process records at the specified slog.Level.
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.logger.level.val.Load() <= int64(slogLevelToVelo(level))
}

// Handle processes a slog.Record, converting it into a Velo log entry.
func (h *SlogHandler) Handle(_ context.Context, r slog.Record) error {
	level := slogLevelToVelo(r.Level)

	// Use a pooled buffer for fields to reduce allocations if we were doing complex formatting,
	// but here we are passing Fields to LogFields.
	// LogFields itself might allocate if we pass a slice.

	fields := make([]Field, 0, r.NumAttrs()+len(h.attrs))
	fields = append(fields, h.attrs...)

	r.Attrs(func(a slog.Attr) bool {
		fields = append(fields, slogAttrToField(a, h.group))
		return true
	})

	h.logger.LogFields(level, r.Message, fields...)
	return nil
}

// WithAttrs creates a new SlogHandler that includes the specified attributes in every log entry.
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	newAttrs := make([]Field, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	for _, a := range attrs {
		newAttrs = append(newAttrs, slogAttrToField(a, h.group))
	}
	return &SlogHandler{
		logger: h.logger,
		attrs:  newAttrs,
		group:  h.group,
	}
}

// WithGroup creates a new SlogHandler that prefixes all subsequent attribute keys with the specified group name.
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	newGroup := h.group
	if newGroup != "" {
		newGroup += "." + name
	} else {
		newGroup = name
	}
	return &SlogHandler{
		logger: h.logger,
		attrs:  h.attrs,
		group:  newGroup,
	}
}

func slogLevelToVelo(l slog.Level) Level {
	switch {
	case l >= slog.LevelError:
		return ErrorLevel
	case l >= slog.LevelWarn:
		return WarnLevel
	case l >= slog.LevelInfo:
		return InfoLevel
	default:
		return DebugLevel
	}
}

func slogAttrToField(a slog.Attr, group string) Field {
	key := a.Key
	if group != "" {
		key = group + "." + key
	}

	switch a.Value.Kind() {
	case slog.KindString:
		return String(key, a.Value.String())
	case slog.KindInt64:
		return Int64(key, a.Value.Int64())
	case slog.KindBool:
		return Bool(key, a.Value.Bool())
	case slog.KindDuration:
		return Int64(key, int64(a.Value.Duration()))
	case slog.KindTime:
		return String(key, a.Value.Time().String())
	case slog.KindAny:
		if err, ok := a.Value.Any().(error); ok {
			return Err(err)
		}
		return Any(key, a.Value.Any())
	default:
		return Any(key, a.Value.Any())
	}
}
