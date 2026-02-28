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
	"sync"
	"time"
)

// Entry encapsulates all the data associated with a single log event.
//
// It holds the timestamp, message, level, caller information, and any attached
// fields. The Logger populates this struct before passing it to the Formatter.
//
// Performance Note: The library pools Entry objects to eliminate allocations
// during high throughput logging. The Logger retrieves an Entry from the pool,
// populates it, formats it, and then immediately returns it to the pool.
type Entry struct {
	Time           time.Time
	Fields         []any
	TypedFields    []Field
	PreEncodedJSON []byte
	Stack          []uintptr
	Message        string
	Prefix         string
	Caller         string
	TimeFormat     string
	Formatter      Formatter
	Level          Level
}

var _entryPool = sync.Pool{
	New: func() any {
		return &Entry{
			Fields:      make([]any, 0, 20),
			TypedFields: make([]Field, 0, 20),
		}
	},
}

func getEntry() *Entry {
	return _entryPool.Get().(*Entry)
}

func putEntry(e *Entry) {
	// Reset the entry for reuse.
	e.Fields = e.Fields[:0]
	e.TypedFields = e.TypedFields[:0]
	e.PreEncodedJSON = nil
	e.Stack = e.Stack[:0]
	_entryPool.Put(e)
}
