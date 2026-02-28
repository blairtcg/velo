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
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/cpu"
)

var (
	_defaultLogger     atomic.Pointer[Logger]
	_defaultLoggerOnce sync.Once
)

// Default returns the global default Logger instance.
//
// It initializes a new Logger writing to standard error with timestamp reporting
// enabled if one does not already exist. Use this for simple applications or
// quick debugging where dependency injection is unnecessary.
func Default() *Logger {
	dl := _defaultLogger.Load()
	if dl == nil {
		_defaultLoggerOnce.Do(func() {
			l := NewWithOptions(os.Stderr, Options{ReportTimestamp: true})
			_defaultLogger.CompareAndSwap(nil, l)
		})
		dl = _defaultLogger.Load()
	}
	return dl
}

// SetDefault replaces the global default Logger with the provided instance.
//
// It safely swaps the underlying pointer and closes the previous Logger if one
// existed. Use this during application startup to configure the global logging
// behavior for all subsequent calls to package level functions.
func SetDefault(logger *Logger) {
	old := _defaultLogger.Swap(logger)
	if old != nil {
		old.Close()
	}
}

// New constructs a new Logger writing to the provided io.Writer.
//
// It applies the default Options. If the provided writer is nil, it defaults
// to standard error. This provides a quick way to instantiate a Logger without
// configuring advanced features.
func New(w io.Writer) *Logger {
	return NewWithOptions(w, Options{})
}

type loggerAlloc struct {
	logger Logger
	level  levelState
	out    syncWriter
	config loggerConfig
}

// NewWithOptions constructs a new Logger using the specified Options.
//
// It provides full control over the Logger's behavior, including asynchronous
// writing, formatting, and field extraction. If the provided writer is nil, it
// defaults to standard error. This is the recommended way to instantiate a
// Logger for production applications.
func NewWithOptions(w io.Writer, o Options) *Logger {
	if o.BufferSize == 0 {
		o.BufferSize = 8192
	}
	if w == nil {
		w = os.Stderr
	}

	alloc := &loggerAlloc{}
	l := &alloc.logger

	alloc.config = loggerConfig{
		prefix:           o.Prefix,
		timeFunc:         o.TimeFunction,
		timeFormat:       o.TimeFormat,
		callerOffset:     o.CallerOffset,
		callerFormatter:  o.CallerFormatter,
		formatter:        o.Formatter,
		contextExtractor: o.ContextExtractor,
		reportTimestamp:  o.ReportTimestamp,
		reportCaller:     o.ReportCaller,
		reportStacktrace: o.ReportStacktrace,
	}

	if alloc.config.callerFormatter == nil {
		alloc.config.callerFormatter = ShortCallerFormatter
	}
	if alloc.config.timeFormat == "" {
		alloc.config.timeFormat = DefaultTimeFormat
	}

	l.level = &alloc.level
	l.fields = o.Fields

	if o.Async {
		l.worker = newWorker(w, o.BufferSize, o.OverflowStrategy)
	} else {
		alloc.out.out = w
		l.out = &alloc.out
	}

	l.level.val.Store(int64(o.Level))
	l.config.Store(&alloc.config)

	return l
}

type levelState struct {
	_   cpu.CacheLinePad
	val atomic.Int64
	_   cpu.CacheLinePad
}

type syncWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func (s *syncWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	n, err = s.out.Write(p)
	s.mu.Unlock()
	return
}

func (s *syncWriter) Sync() error {
	if syncer, ok := s.out.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

type loggerConfig struct {
	prefix           string
	timeFunc         TimeFunction
	timeFormat       string
	callerOffset     int
	callerFormatter  CallerFormatter
	formatter        Formatter
	contextExtractor ContextExtractor
	reportTimestamp  bool
	reportCaller     bool
	reportStacktrace bool
}

// Logger provides fast, leveled, and structured logging.
//
// It is designed for contexts where every microsecond and allocation matters.
// The Logger supports both synchronous and asynchronous writing, zero allocation
// field logging, and dynamic context extraction. While a global default Logger
// exists, instantiating local instances via NewWithOptions and injecting them
// as dependencies avoids global state and improves testability. All methods
// are safe for concurrent use.
type Logger struct {
	level *levelState
	_     cpu.CacheLinePad

	closed atomic.Uint32
	_      cpu.CacheLinePad

	config atomic.Pointer[loggerConfig]

	fields         []any
	typedFields    []Field
	preEncodedJSON []byte

	worker *worker
	out    *syncWriter

	sampler *sampler
}

// Close stops the background worker and flushes all remaining log entries.
//
// You must call Close before your application exits to ensure no asynchronous
// logs are lost. It safely decrements the worker reference count and stops the
// worker when the count reaches zero. Calling Close on a synchronous Logger
// has no effect.
func (l *Logger) Close() {
	if l.closed.CompareAndSwap(0, 1) {
		if l.worker != nil {
			if l.worker.refCount.Add(-1) == 0 {
				l.worker.stop()
			}
		}
	}
}

// Sync flushes any buffered log entries to the underlying writer.
//
// It delegates to the worker's sync method for asynchronous loggers, or calls
// Sync on the underlying io.Writer if it implements the interface. Use this
// to ensure critical logs are written immediately.
func (l *Logger) Sync() error {
	if l.worker != nil {
		return l.worker.sync()
	}
	if l.out != nil {
		return l.out.Sync()
	}
	return nil
}

func (l *Logger) submit(b *buffer) {
	if l.worker != nil {
		l.worker.submit(b)
	} else if l.out != nil {
		l.out.Write(b.B)
		putBuffer(b)
	} else {
		putBuffer(b)
	}
}

// LogContext writes a message with loosely typed key-value pairs at the specified level.
//
// It extracts context specific fields if a ContextExtractor is configured.
//
// Performance Note: This method iterates over the key-value pairs to check for
// errors and capture stack traces. This adds a slight type assertion overhead.
// For absolute maximum performance and zero allocations, use the strongly typed
// LogContextFields method instead.
func (l *Logger) LogContext(ctx context.Context, level Level, msg string, keyvals ...any) {
	if l.level.val.Load() > int64(level) {
		return
	}
	l.logContext(ctx, level, msg, keyvals)
}

func (l *Logger) logContext(ctx context.Context, level Level, msg string, keyvals []any) {
	cfg := l.config.Load()

	var t time.Time
	if cfg.reportTimestamp {
		if cfg.timeFunc != nil {
			t = cfg.timeFunc(time.Now())
		} else {
			t = time.Now()
		}
	}

	if l.sampler != nil && !l.sampler.check(level, msg, t) {
		return
	}

	var ctxFields []Field
	if cfg.contextExtractor != nil && ctx != nil {
		ctxFields = cfg.contextExtractor(ctx)
	}

	if cfg.reportStacktrace || cfg.reportCaller {
		l.logWithEntry(level, msg, keyvals, nil, ctxFields, cfg, t)
		return
	}

	// Fast path: direct formatting
	b := getBuffer()

	if cfg.formatter == JSONFormatter {
		formatLogJSON(b, l, cfg, level, msg, keyvals, nil, ctxFields, t)
	} else {
		formatLogText(b, l, cfg, level, msg, keyvals, nil, ctxFields, t)
	}

	l.submit(b)

	if level == PanicLevel {
		l.Sync()
		panic(msg)
	}

	if level == FatalLevel {
		flushAllWorkers()
		os.Exit(1)
	}
}

// LogContextFields writes a message with strongly typed fields at the specified level.
//
// It extracts context specific fields if a ContextExtractor is configured. This
// method guarantees zero allocations on the hot path, making it ideal for
// extreme high throughput, latency critical applications.
func (l *Logger) LogContextFields(ctx context.Context, level Level, msg string, fields ...Field) {
	if l.level.val.Load() > int64(level) {
		return
	}
	l.logContextFields(ctx, level, msg, fields)
}

func (l *Logger) logContextFields(ctx context.Context, level Level, msg string, fields []Field) {
	cfg := l.config.Load()

	var t time.Time
	if cfg.reportTimestamp {
		if cfg.timeFunc != nil {
			t = cfg.timeFunc(time.Now())
		} else {
			t = time.Now()
		}
	}

	if l.sampler != nil && !l.sampler.check(level, msg, t) {
		return
	}

	var ctxFields []Field
	if cfg.contextExtractor != nil && ctx != nil {
		ctxFields = cfg.contextExtractor(ctx)
	}

	if cfg.reportStacktrace || cfg.reportCaller {
		l.logWithEntry(level, msg, nil, fields, ctxFields, cfg, t)
		return
	}

	// Fast path: direct formatting
	b := getBuffer()

	if cfg.formatter == JSONFormatter {
		formatLogJSON(b, l, cfg, level, msg, nil, fields, ctxFields, t)
	} else {
		formatLogText(b, l, cfg, level, msg, nil, fields, ctxFields, t)
	}

	l.submit(b)

	if level == PanicLevel {
		l.Sync()
		panic(msg)
	}

	if level == FatalLevel {
		flushAllWorkers()
		os.Exit(1)
	}
}

// With creates a child Logger that includes the provided loosely typed key-value pairs.
//
// It copies the parent's configuration and appends the new fields. If the
// JSONFormatter is active, it encodes the fields early to avoid redundant
// serialization on every log call. Use this to attach contextual data to a
// Logger for a specific scope or request.
func (l *Logger) With(keyvals ...any) *Logger {
	if len(keyvals) == 0 {
		return l
	}
	newFields := make([]any, len(l.fields), len(l.fields)+len(keyvals))
	copy(newFields, l.fields)
	newFields = append(newFields, keyvals...)

	nl := &Logger{
		fields:      newFields,
		typedFields: l.typedFields,
		worker:      l.worker,
		out:         l.out,
		level:       l.level,
		sampler:     l.sampler,
	}
	nl.config.Store(l.config.Load())

	// Pre-encode JSON fields if using JSONFormatter
	cfg := l.config.Load()
	if cfg.formatter == JSONFormatter {
		b := getBuffer()
		if len(l.preEncodedJSON) > 0 {
			b.Write(l.preEncodedJSON)
		}
		for i := 0; i < len(keyvals); i += 2 {
			if i+1 < len(keyvals) {
				encodeKeyValToJSON(b, keyvals[i], keyvals[i+1], true)
			}
		}
		nl.preEncodedJSON = make([]byte, len(b.B))
		copy(nl.preEncodedJSON, b.B)
		putBuffer(b)
	}

	if l.worker != nil {
		l.worker.refCount.Add(1)
	}
	return nl
}

// WithFields creates a child Logger that includes the provided strongly typed fields.
//
// It copies the parent's configuration and appends the new fields. If the
// JSONFormatter is active, it encodes the fields early to avoid redundant
// serialization on every log call. This provides the highest performance when
// attaching contextual data to a Logger.
func (l *Logger) WithFields(fields ...Field) *Logger {
	if len(fields) == 0 {
		return l
	}
	newFields := make([]Field, len(l.typedFields), len(l.typedFields)+len(fields))
	copy(newFields, l.typedFields)
	newFields = append(newFields, fields...)

	nl := &Logger{
		fields:      l.fields,
		typedFields: newFields,
		worker:      l.worker,
		out:         l.out,
		level:       l.level,
		sampler:     l.sampler,
	}
	nl.config.Store(l.config.Load())

	// Pre-encode JSON fields if using JSONFormatter
	cfg := l.config.Load()
	if cfg.formatter == JSONFormatter {
		b := getBuffer()
		if len(l.preEncodedJSON) > 0 {
			b.Write(l.preEncodedJSON)
		}
		for i := 0; i < len(fields); i++ {
			encodeFieldToJSON(b, &fields[i], cfg.timeFormat, true)
		}
		nl.preEncodedJSON = make([]byte, len(b.B))
		copy(nl.preEncodedJSON, b.B)
		putBuffer(b)
	}

	if l.worker != nil {
		l.worker.refCount.Add(1)
	}
	return nl
}

// WithPrefix creates a child Logger that prepends the specified prefix to all messages.
//
// It copies the parent's configuration and updates the prefix. Use this to
// visually group logs from a specific component or subsystem.
func (l *Logger) WithPrefix(prefix string) *Logger {
	nl := l.With()
	nl.SetPrefix(prefix)
	return nl
}

// Logf formats and writes a message at the specified level.
//
// It uses fmt.Sprintf to construct the message. This incurs allocation and
// formatting overhead. Avoid using this in performance critical paths.
func (l *Logger) Logf(level Level, format string, args ...any) {
	l.Log(level, fmt.Sprintf(format, args...))
}

// SetLevel changes the minimum logging level for this Logger dynamically.
//
// It uses atomic operations to ensure thread safety. The Logger discards any
// messages below this level. Use this to adjust verbosity at runtime without
// restarting the application.
func (l *Logger) SetLevel(level Level) {
	l.level.val.Store(int64(level))
}

// SetReportTimestamp toggles the inclusion of timestamps in log entries.
//
// It safely updates the Logger's configuration. Disabling timestamps can
// slightly improve performance and reduce log volume if your log aggregator
// already assigns timestamps.
func (l *Logger) SetReportTimestamp(report bool) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.reportTimestamp = report
	l.config.Store(&newCfg)
}

// SetReportCaller toggles the inclusion of the caller's file and line number.
//
// It safely updates the Logger's configuration.
//
// Performance Note: Enabling this feature incurs a significant performance
// penalty because it requires unwinding the stack using runtime.Caller. Use
// with caution in high throughput, latency critical paths.
func (l *Logger) SetReportCaller(report bool) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.reportCaller = report
	l.config.Store(&newCfg)
}

// SetReportStacktrace toggles the automatic capture of stack traces for errors.
//
// It safely updates the Logger's configuration. When enabled, the Logger
// captures a stack trace whenever it writes an entry at ErrorLevel or higher,
// or when an error field is present.
//
// Performance Note: Capturing stack traces incurs a significant performance
// penalty. Use this feature primarily for debugging or in environments where
// error rates are low.
func (l *Logger) SetReportStacktrace(report bool) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.reportStacktrace = report
	l.config.Store(&newCfg)
}

// SetPrefix changes the prefix prepended to all messages for this Logger.
//
// It safely updates the Logger's configuration. Use this to dynamically label
// logs from a specific component.
func (l *Logger) SetPrefix(prefix string) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.prefix = prefix
	l.config.Store(&newCfg)
}

// SetTimeFormat changes the timestamp format string.
//
// It safely updates the Logger's configuration. The format string must conform
// to the layout expected by the standard time package.
func (l *Logger) SetTimeFormat(format string) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.timeFormat = format
	l.config.Store(&newCfg)
}

// SetTimeFunction changes the function used to generate timestamps.
//
// It safely updates the Logger's configuration. Use this to inject a custom
// clock for testing or to apply specific timezone adjustments.
func (l *Logger) SetTimeFunction(f TimeFunction) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.timeFunc = f
	l.config.Store(&newCfg)
}

// SetFormatter changes the Formatter used to serialize log entries.
//
// It safely updates the Logger's configuration. You can switch between built in
// formatters like JSONFormatter and TextFormatter, or provide a custom
// implementation.
func (l *Logger) SetFormatter(f Formatter) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.formatter = f
	l.config.Store(&newCfg)
}

// SetCallerFormatter changes the function used to format caller location data.
//
// It safely updates the Logger's configuration. Use this to customize how file
// paths and line numbers appear in your logs.
func (l *Logger) SetCallerFormatter(f CallerFormatter) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.callerFormatter = f
	l.config.Store(&newCfg)
}

// SetCallerOffset changes the number of stack frames to skip when identifying the caller.
//
// It safely updates the Logger's configuration. Increase this value if you wrap
// the Logger in custom helper functions to ensure the reported caller reflects
// the actual log origin.
func (l *Logger) SetCallerOffset(offset int) {
	cfg := l.config.Load()
	newCfg := *cfg
	newCfg.callerOffset = offset
	l.config.Store(&newCfg)
}

// Debug writes a message at DebugLevel with loosely typed key-value pairs.
func (l *Logger) Debug(msg string, keyvals ...any) { l.Log(DebugLevel, msg, keyvals...) }

// Info writes a message at InfoLevel with loosely typed key-value pairs.
func (l *Logger) Info(msg string, keyvals ...any) { l.Log(InfoLevel, msg, keyvals...) }

// Warn writes a message at WarnLevel with loosely typed key-value pairs.
func (l *Logger) Warn(msg string, keyvals ...any) { l.Log(WarnLevel, msg, keyvals...) }

// Error writes a message at ErrorLevel with loosely typed key-value pairs.
func (l *Logger) Error(msg string, keyvals ...any) { l.Log(ErrorLevel, msg, keyvals...) }

// Panic writes a message at PanicLevel with loosely typed key-value pairs, then panics.
func (l *Logger) Panic(msg string, keyvals ...any) { l.Log(PanicLevel, msg, keyvals...) }

// Fatal writes a message at FatalLevel with loosely typed key-value pairs, then calls os.Exit(1).
func (l *Logger) Fatal(msg string, keyvals ...any) { l.Log(FatalLevel, msg, keyvals...) }

// Print writes a message with no level and loosely typed key-value pairs.
func (l *Logger) Print(msg string, keyvals ...any) { l.Log(noLevel, msg, keyvals...) }

// Debugf formats and writes a message at DebugLevel.
func (l *Logger) Debugf(format string, args ...any) { l.Log(DebugLevel, fmt.Sprintf(format, args...)) }

// Infof formats and writes a message at InfoLevel.
func (l *Logger) Infof(format string, args ...any) { l.Log(InfoLevel, fmt.Sprintf(format, args...)) }

// Warnf formats and writes a message at WarnLevel.
func (l *Logger) Warnf(format string, args ...any) { l.Log(WarnLevel, fmt.Sprintf(format, args...)) }

// Errorf formats and writes a message at ErrorLevel.
func (l *Logger) Errorf(format string, args ...any) { l.Log(ErrorLevel, fmt.Sprintf(format, args...)) }

// Panicf formats and writes a message at PanicLevel, then panics.
func (l *Logger) Panicf(format string, args ...any) { l.Log(PanicLevel, fmt.Sprintf(format, args...)) }

// Fatalf formats and writes a message at FatalLevel, then calls os.Exit(1).
func (l *Logger) Fatalf(format string, args ...any) { l.Log(FatalLevel, fmt.Sprintf(format, args...)) }

// Printf formats and writes a message with no level.
func (l *Logger) Printf(format string, args ...any) { l.Log(noLevel, fmt.Sprintf(format, args...)) }

// DebugFields writes a message at DebugLevel with strongly typed fields, guaranteeing zero allocations.
func (l *Logger) DebugFields(msg string, fields ...Field) { l.LogFields(DebugLevel, msg, fields...) }

// InfoFields writes a message at InfoLevel with strongly typed fields, guaranteeing zero allocations.
func (l *Logger) InfoFields(msg string, fields ...Field) { l.LogFields(InfoLevel, msg, fields...) }

// WarnFields writes a message at WarnLevel with strongly typed fields, guaranteeing zero allocations.
func (l *Logger) WarnFields(msg string, fields ...Field) { l.LogFields(WarnLevel, msg, fields...) }

// ErrorFields writes a message at ErrorLevel with strongly typed fields, guaranteeing zero allocations.
func (l *Logger) ErrorFields(msg string, fields ...Field) { l.LogFields(ErrorLevel, msg, fields...) }

// PanicFields writes a message at PanicLevel with strongly typed fields, guaranteeing zero allocations, then panics.
func (l *Logger) PanicFields(msg string, fields ...Field) { l.LogFields(PanicLevel, msg, fields...) }

// FatalFields writes a message at FatalLevel with strongly typed fields, guaranteeing zero allocations, then calls os.Exit(1).
func (l *Logger) FatalFields(msg string, fields ...Field) { l.LogFields(FatalLevel, msg, fields...) }

// getCaller identifies the file, line, and function name of the calling code.
//
// It uses runtime.Caller for maximum performance, avoiding the heavy allocation
// overhead associated with runtime.CallersFrames.
func (l *Logger) getCaller(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0, ""
	}
	fn := ""
	if f := runtime.FuncForPC(pc); f != nil {
		fn = f.Name()
	}
	return file, line, fn
}

// Log writes a message with loosely typed key-value pairs at the specified level.
//
// Performance Note: This method iterates over the key-value pairs to check for
// errors and capture stack traces. This adds a slight type assertion overhead.
// For absolute maximum performance and zero allocations, use the strongly typed
// LogFields method instead.
func (l *Logger) Log(level Level, msg string, keyvals ...any) {
	if l.level.val.Load() > int64(level) {
		return
	}
	l.log(level, msg, keyvals)
}

func (l *Logger) log(level Level, msg string, keyvals []any) {
	cfg := l.config.Load()

	var t time.Time
	if cfg.reportTimestamp {
		if cfg.timeFunc != nil {
			t = cfg.timeFunc(time.Now())
		} else {
			t = time.Now()
		}
	}

	if l.sampler != nil && !l.sampler.check(level, msg, t) {
		return
	}

	// If we need stacktrace or caller, we still need Entry or similar structure,
	// OR we can just handle them here.
	// For maximum performance on the hot path (no stack/caller), we skip Entry.

	if cfg.reportStacktrace || cfg.reportCaller {
		// Fallback to full Entry path for complex cases
		l.logWithEntry(level, msg, keyvals, nil, nil, cfg, t)
		return
	}

	// Fast path: direct formatting
	b := getBuffer()

	if cfg.formatter == JSONFormatter {
		formatLogJSON(b, l, cfg, level, msg, keyvals, nil, nil, t)
	} else {
		formatLogText(b, l, cfg, level, msg, keyvals, nil, nil, t)
	}

	l.submit(b)

	if level == PanicLevel {
		l.Sync()
		panic(msg)
	}

	if level == FatalLevel {
		flushAllWorkers()
		os.Exit(1)
	}
}

func (l *Logger) logWithEntry(level Level, msg string, keyvals []any, typedFields []Field, ctxFields []Field, cfg *loggerConfig, t time.Time) {
	e := getEntry()
	e.Level = level
	e.Time = t
	e.Message = msg
	e.Prefix = cfg.prefix
	e.Formatter = cfg.formatter
	e.TimeFormat = cfg.timeFormat

	// append logger fields
	if cfg.formatter == JSONFormatter && (len(l.preEncodedJSON) > 0 || (len(l.fields) == 0 && len(l.typedFields) == 0)) {
		e.PreEncodedJSON = l.preEncodedJSON
	} else {
		if len(l.fields) > 0 {
			e.Fields = append(e.Fields, l.fields...)
		}
		if len(l.typedFields) > 0 {
			e.TypedFields = append(e.TypedFields, l.typedFields...)
		}
	}

	// append log-specific fields
	if len(keyvals) > 0 {
		e.Fields = append(e.Fields, keyvals...)
	}
	if len(ctxFields) > 0 {
		e.TypedFields = append(e.TypedFields, ctxFields...)
	}
	if len(typedFields) > 0 {
		e.TypedFields = append(e.TypedFields, typedFields...)
	}

	if cfg.reportStacktrace {
		hasErr := level >= ErrorLevel

		if !hasErr {
			for i := 0; i < len(keyvals); i++ {
				if _, ok := keyvals[i].(error); ok {
					hasErr = true
					break
				}
			}
			for _, f := range ctxFields {
				if f.Type == ErrorType {
					hasErr = true
					break
				}
			}
			for _, f := range typedFields {
				if f.Type == ErrorType {
					hasErr = true
					break
				}
			}
		}

		if hasErr {
			var pcs [32]uintptr
			n := runtime.Callers(4, pcs[:]) // +1 for logWithEntry
			e.Stack = append(e.Stack[:0], pcs[:n]...)
		}
	}

	if cfg.reportCaller {
		file, line, fn := l.getCaller(cfg.callerOffset + 4) // +1 for logWithEntry
		if file != "" && cfg.callerFormatter != nil {
			e.Caller = cfg.callerFormatter(file, line, fn)
		}
	}

	b := getBuffer()
	formatEntry(b, e)
	putEntry(e)

	l.submit(b)

	if level == PanicLevel {
		l.Sync()
		panic(msg)
	}

	if level == FatalLevel {
		flushAllWorkers()
		os.Exit(1)
	}
}

// LogFields writes a message with strongly typed fields at the specified level.
//
// This method guarantees zero allocations on the hot path, making it ideal for
// extreme high throughput, latency critical applications.
func (l *Logger) LogFields(level Level, msg string, fields ...Field) {
	if l.level.val.Load() > int64(level) {
		return
	}
	l.logFields(level, msg, fields)
}

func (l *Logger) logFields(level Level, msg string, fields []Field) {
	cfg := l.config.Load()

	var t time.Time
	if cfg.reportTimestamp {
		if cfg.timeFunc != nil {
			t = cfg.timeFunc(time.Now())
		} else {
			t = time.Now()
		}
	}

	if l.sampler != nil && !l.sampler.check(level, msg, t) {
		return
	}

	if cfg.reportStacktrace || cfg.reportCaller {
		l.logWithEntry(level, msg, nil, fields, nil, cfg, t)
		return
	}

	// Fast path: direct formatting
	b := getBuffer()

	if cfg.formatter == JSONFormatter {
		formatLogJSON(b, l, cfg, level, msg, nil, fields, nil, t)
	} else {
		formatLogText(b, l, cfg, level, msg, nil, fields, nil, t)
	}

	l.submit(b)

	if level == PanicLevel {
		l.Sync()
		panic(msg)
	}

	if level == FatalLevel {
		flushAllWorkers()
		os.Exit(1)
	}
}

// Global functions

// SetReportTimestamp toggles timestamp reporting for the global default Logger.
func SetReportTimestamp(report bool) { Default().SetReportTimestamp(report) }

// SetReportCaller toggles caller location reporting for the global default Logger.
func SetReportCaller(report bool) { Default().SetReportCaller(report) }

// SetLevel changes the minimum logging level for the global default Logger.
func SetLevel(level Level) { Default().SetLevel(level) }

// SetTimeFormat changes the timestamp format string for the global default Logger.
func SetTimeFormat(format string) { Default().SetTimeFormat(format) }

// SetTimeFunction changes the timestamp generation function for the global default Logger.
func SetTimeFunction(f TimeFunction) { Default().SetTimeFunction(f) }

// SetFormatter changes the serialization Formatter for the global default Logger.
func SetFormatter(f Formatter) { Default().SetFormatter(f) }

// SetCallerFormatter changes the caller formatting function for the global default Logger.
func SetCallerFormatter(f CallerFormatter) { Default().SetCallerFormatter(f) }

// SetCallerOffset changes the stack frame offset for the global default Logger.
func SetCallerOffset(offset int) { Default().SetCallerOffset(offset) }

// SetPrefix changes the message prefix for the global default Logger.
func SetPrefix(prefix string) { Default().SetPrefix(prefix) }

// With creates a child of the global default Logger with the provided loosely typed fields.
func With(keyvals ...any) *Logger { return Default().With(keyvals...) }

// WithFields creates a child of the global default Logger with the provided strongly typed fields.
func WithFields(fields ...Field) *Logger { return Default().WithFields(fields...) }

// WithPrefix creates a child of the global default Logger with the specified prefix.
func WithPrefix(prefix string) *Logger { return Default().WithPrefix(prefix) }

// Log writes a message to the global default Logger at the specified level.
func Log(level Level, msg string, keyvals ...any) { Default().Log(level, msg, keyvals...) }

// Debug writes a message to the global default Logger at DebugLevel.
func Debug(msg string, keyvals ...any) { Default().Log(DebugLevel, msg, keyvals...) }

// Info writes a message to the global default Logger at InfoLevel.
func Info(msg string, keyvals ...any) { Default().Log(InfoLevel, msg, keyvals...) }

// Warn writes a message to the global default Logger at WarnLevel.
func Warn(msg string, keyvals ...any) { Default().Log(WarnLevel, msg, keyvals...) }

// Error writes a message to the global default Logger at ErrorLevel.
func Error(msg string, keyvals ...any) { Default().Log(ErrorLevel, msg, keyvals...) }

// Panic writes a message to the global default Logger at PanicLevel, then panics.
func Panic(msg string, keyvals ...any) { Default().Log(PanicLevel, msg, keyvals...) }

// Fatal writes a message to the global default Logger at FatalLevel, then calls os.Exit(1).
func Fatal(msg string, keyvals ...any) { Default().Log(FatalLevel, msg, keyvals...) }

// Print writes a message to the global default Logger with no level.
func Print(msg string, keyvals ...any) { Default().Log(noLevel, msg, keyvals...) }

// Logf formats and writes a message to the global default Logger at the specified level.
func Logf(level Level, format string, args ...any) { Default().Logf(level, format, args...) }

// Debugf formats and writes a message to the global default Logger at DebugLevel.
func Debugf(format string, args ...any) { Default().Debugf(format, args...) }

// Infof formats and writes a message to the global default Logger at InfoLevel.
func Infof(format string, args ...any) { Default().Infof(format, args...) }

// Warnf formats and writes a message to the global default Logger at WarnLevel.
func Warnf(format string, args ...any) { Default().Warnf(format, args...) }

// Errorf formats and writes a message to the global default Logger at ErrorLevel.
func Errorf(format string, args ...any) { Default().Errorf(format, args...) }

// Panicf formats and writes a message to the global default Logger at PanicLevel, then panics.
func Panicf(format string, args ...any) { Default().Panicf(format, args...) }

// Fatalf formats and writes a message to the global default Logger at FatalLevel, then calls os.Exit(1).
func Fatalf(format string, args ...any) { Default().Fatalf(format, args...) }

// Printf formats and writes a message to the global default Logger with no level.
func Printf(format string, args ...any) { Default().Printf(format, args...) }

// DebugFields writes a message to the global default Logger at DebugLevel with strongly typed fields.
func DebugFields(msg string, fields ...Field) { Default().LogFields(DebugLevel, msg, fields...) }

// InfoFields writes a message to the global default Logger at InfoLevel with strongly typed fields.
func InfoFields(msg string, fields ...Field) { Default().LogFields(InfoLevel, msg, fields...) }

// WarnFields writes a message to the global default Logger at WarnLevel with strongly typed fields.
func WarnFields(msg string, fields ...Field) { Default().LogFields(WarnLevel, msg, fields...) }

// ErrorFields writes a message to the global default Logger at ErrorLevel with strongly typed fields.
func ErrorFields(msg string, fields ...Field) { Default().LogFields(ErrorLevel, msg, fields...) }

// PanicFields writes a message to the global default Logger at PanicLevel with strongly typed fields, then panics.
func PanicFields(msg string, fields ...Field) { Default().LogFields(PanicLevel, msg, fields...) }

// FatalFields writes a message to the global default Logger at FatalLevel with strongly typed fields, then calls os.Exit(1).
func FatalFields(msg string, fields ...Field) { Default().LogFields(FatalLevel, msg, fields...) }
