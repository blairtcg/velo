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
	"fmt"
	"io"
	"path/filepath"
	"time"
)

// Formatter defines the serialization format for log entries.
type Formatter int

const (
	// TextFormatter serializes log entries as human readable, colorized text.
	TextFormatter Formatter = iota
	// JSONFormatter serializes log entries as structured JSON.
	JSONFormatter
)

// OverflowStrategy dictates how an asynchronous Logger behaves when its internal ring buffer fills up.
type OverflowStrategy int

const (
	// OverflowSync forces the Logger to write directly to the underlying writer,
	// bypassing the full buffer. This guarantees no logs are lost but temporarily
	// introduces synchronous blocking.
	OverflowSync OverflowStrategy = iota
	// OverflowDrop discards new log entries until space becomes available in the
	// buffer. This prioritizes application performance over log completeness.
	OverflowDrop
	// OverflowBlock pauses the calling goroutine until the background worker
	// frees up space in the buffer. This guarantees no logs are lost but can
	// severely impact application latency.
	OverflowBlock
)

// TimeFunction defines a custom hook for generating or modifying timestamps.
type TimeFunction func(time.Time) time.Time

// CallerFormatter defines a custom hook for formatting file and line number information.
type CallerFormatter func(file string, line int, funcName string) string

// ShortCallerFormatter returns the file name and line number (e.g., "logger.go:42").
func ShortCallerFormatter(file string, line int, funcName string) string {
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// LongCallerFormatter returns the absolute file path and line number (e.g., "/path/to/logger.go:42").
func LongCallerFormatter(file string, line int, funcName string) string {
	return fmt.Sprintf("%s:%d", file, line)
}

// ContextExtractor defines a custom hook for extracting strongly typed fields from a context.Context.
type ContextExtractor func(context.Context) []Field

// Options configures the behavior of a new Logger.
type Options struct {
	// Level sets the minimum logging priority. The Logger discards entries below this level.
	Level Level

	// Output specifies the destination for log data.
	// Deprecated: Pass the io.Writer directly to NewWithOptions instead.
	Output io.Writer

	// BufferSize defines the capacity of the internal ring buffer for asynchronous loggers.
	// It must be a power of 2. It defaults to 8192.
	BufferSize int

	// OverflowStrategy dictates behavior when the asynchronous buffer fills up.
	// It defaults to OverflowSync.
	OverflowStrategy OverflowStrategy

	// ReportTimestamp includes a timestamp in every log entry.
	ReportTimestamp bool

	// TimeFormat specifies the layout string for timestamps.
	// It defaults to DefaultTimeFormat.
	TimeFormat string

	// TimeFunction provides a custom hook for generating timestamps.
	// It defaults to time.Now.
	TimeFunction TimeFunction

	// ReportCaller includes the calling file and line number in every log entry.
	// Performance Note: Enabling this incurs a significant performance penalty.
	ReportCaller bool

	// CallerOffset adjusts the stack frame depth when identifying the caller.
	// Use this if you wrap the Logger in custom helper functions.
	CallerOffset int

	// CallerFormatter provides a custom hook for formatting caller information.
	// It defaults to ShortCallerFormatter.
	CallerFormatter CallerFormatter

	// ReportStacktrace includes a full stack trace for entries at ErrorLevel or higher.
	// Performance Note: Enabling this incurs a significant performance penalty on errors.
	ReportStacktrace bool

	// Prefix prepends a static string to every log message.
	Prefix string

	// Fields attaches default, loosely typed key-value pairs to every log entry.
	Fields []any

	// Formatter dictates how the Logger serializes entries (e.g., TextFormatter or JSONFormatter).
	// It defaults to TextFormatter.
	Formatter Formatter

	// ContextExtractor provides a custom hook to pull fields from a context.Context.
	ContextExtractor ContextExtractor

	// Async enables the background worker, routing logs through a lock free ring buffer.
	Async bool
}

// DefaultTimeFormat specifies the standard timestamp layout used when no custom format is provided.
const DefaultTimeFormat = "2006/01/02 15:04:05"
