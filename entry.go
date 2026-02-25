package velo

import (
	"sync"
	"time"
)

// Entry contains the details of a single log event.
//
// It holds the core message, metadata, and structured fields needed
// to format and output the log. The logger builds this struct and 
// passes it to the formatter before writing to the output stream.
type Entry struct {
	Level          Level
	Time           time.Time
	Message        string
	Fields         []any
	TypedFields    []Field
	PreEncodedJSON []byte
	Prefix         string
	Caller         string
	Formatter      Formatter
	TimeFormat     string
	Stack          []uintptr
}

// _entryPool stores reusable log entries.
// It helps reduce memory allocations and keeps the logger fast under heavy load.
var _entryPool = sync.Pool{
	New: func() any {
		return &Entry{
			Fields:      make([]any, 0, 20),
			TypedFields: make([]Field, 0, 20),
		}
	},
}

// getEntry retrieves a clean log entry from the pool.
//
// If the pool is empty, it creates a new entry with preallocated
// slices for your log fields. This prevents unnecessary memory 
// allocations on the hot path.
func getEntry() *Entry {
	return _entryPool.Get
