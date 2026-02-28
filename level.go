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
	"bytes"
	"errors"
	"fmt"
	"sync/atomic"
)

// Level represents a logging priority.
//
// Higher levels indicate more severe conditions. The Logger discards messages
// with a level lower than its configured minimum.
type Level int8

const (
	// DebugLevel designates fine grained informational events that are most
	// useful to debug an application.
	DebugLevel Level = iota - 1
	// InfoLevel designates informational messages that highlight the progress
	// of the application at coarse grained level. This is the default level.
	InfoLevel
	// WarnLevel designates potentially harmful situations.
	WarnLevel
	// ErrorLevel designates error events that might still allow the application
	// to continue running.
	ErrorLevel
	// DPanicLevel designates critical errors. In development, the Logger panics
	// after writing the message.
	DPanicLevel
	// PanicLevel designates severe errors. The Logger panics after writing the
	// message.
	PanicLevel
	// FatalLevel designates very severe error events. The Logger calls os.Exit(1)
	// after writing the message.
	FatalLevel

	noLevel Level = 100
)

// JSONField returns the formatted JSON key-value pair for the level.
//
// It provides a zero allocation string (e.g., `"level":"info"`) for the
// JSONEncoder to use during serialization.
func (l Level) JSONField() string {
	switch l {
	case DebugLevel:
		return `"level":"debug"`
	case InfoLevel:
		return `"level":"info"`
	case WarnLevel:
		return `"level":"warn"`
	case ErrorLevel:
		return `"level":"error"`
	case DPanicLevel:
		return `"level":"dpanic"`
	case PanicLevel:
		return `"level":"panic"`
	case FatalLevel:
		return `"level":"fatal"`
	}
	return fmt.Sprintf(`"level":"%s"`, l.String())
}

// String returns the lowercase ASCII representation of the level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case DPanicLevel:
		return "dpanic"
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

// MarshalText serializes the Level to text.
//
// It returns the lowercase string representation of the level (e.g., "info").
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText deserializes text into a Level.
//
// It accepts lowercase or uppercase string representations (e.g., "info" or
// "INFO"). This facilitates configuring log levels via YAML, TOML, or JSON.
func (l *Level) UnmarshalText(text []byte) error {
	if l == nil {
		return errors.New("can't unmarshal a nil *Level")
	}
	if !l.unmarshalText(text) && !l.unmarshalText(bytes.ToLower(text)) {
		return fmt.Errorf("unrecognized level: %q", text)
	}
	return nil
}

func (l *Level) unmarshalText(text []byte) bool {
	switch string(text) {
	case "debug", "DEBUG":
		*l = DebugLevel
	case "info", "INFO", "": // make the zero value useful
		*l = InfoLevel
	case "warn", "WARN":
		*l = WarnLevel
	case "error", "ERROR":
		*l = ErrorLevel
	case "dpanic", "DPANIC":
		*l = DPanicLevel
	case "panic", "PANIC":
		*l = PanicLevel
	case "fatal", "FATAL":
		*l = FatalLevel
	default:
		return false
	}
	return true
}

// ParseLevel converts a string into a Level.
//
// It accepts lowercase or uppercase string representations. It returns an error
// if the string does not match a known level.
func ParseLevel(text string) (Level, error) {
	var l Level
	err := l.UnmarshalText([]byte(text))
	return l, err
}

// AtomicLevel represents a dynamically adjustable logging level.
//
// It allows you to safely change the log level of a Logger and all its
// descendants at runtime without restarting the application. You must create
// an AtomicLevel using the NewAtomicLevel or NewAtomicLevelAt constructors.
type AtomicLevel struct {
	l *atomic.Int32
}

// NewAtomicLevel initializes an AtomicLevel set to InfoLevel.
func NewAtomicLevel() AtomicLevel {
	lvl := AtomicLevel{l: new(atomic.Int32)}
	lvl.l.Store(int32(InfoLevel))
	return lvl
}

// NewAtomicLevelAt initializes an AtomicLevel set to the specified Level.
func NewAtomicLevelAt(l Level) AtomicLevel {
	a := NewAtomicLevel()
	a.SetLevel(l)
	return a
}

// ParseAtomicLevel converts a string into an AtomicLevel.
//
// It accepts lowercase or uppercase string representations. It returns an error
// if the string does not match a known level.
func ParseAtomicLevel(text string) (AtomicLevel, error) {
	a := NewAtomicLevel()
	l, err := ParseLevel(text)
	if err != nil {
		return a, err
	}

	a.SetLevel(l)
	return a, nil
}

// Enabled determines if the specified level meets or exceeds the current minimum.
func (lvl AtomicLevel) Enabled(l Level) bool {
	return lvl.Level() <= l
}

// Level retrieves the current minimum logging level.
func (lvl AtomicLevel) Level() Level {
	return Level(int8(lvl.l.Load()))
}

// SetLevel updates the minimum logging level safely across all goroutines.
func (lvl AtomicLevel) SetLevel(l Level) {
	lvl.l.Store(int32(l))
}

// String returns the string representation of the current minimum level.
func (lvl AtomicLevel) String() string {
	return lvl.Level().String()
}

// UnmarshalText deserializes text into the AtomicLevel.
//
// It accepts the same string representations as the static Level type.
func (lvl *AtomicLevel) UnmarshalText(text []byte) error {
	if lvl.l == nil {
		lvl.l = &atomic.Int32{}
	}

	var l Level
	if err := l.UnmarshalText(text); err != nil {
		return err
	}

	lvl.SetLevel(l)
	return nil
}

// MarshalText serializes the AtomicLevel to a byte slice.
//
// It returns the lowercase string representation of the current minimum level.
func (lvl AtomicLevel) MarshalText() (text []byte, err error) {
	return lvl.Level().MarshalText()
}
