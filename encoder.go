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

import "time"

// ObjectEncoder defines a strongly typed, encoding agnostic interface for adding fields to an object.
//
// It provides methods for appending primitive types and nested structures.
// Implementations of this interface (like the internal JSON encoder) handle the
// actual serialization. This design guarantees zero allocations when logging
// complex objects.
type ObjectEncoder interface {
	AddString(key, value string)
	AddInt(key string, value int)
	AddInt64(key string, value int64)
	AddBool(key string, value bool)
	AddFloat64(key string, value float64)
	AddTime(key string, value time.Time)
	AddDuration(key string, value time.Duration)
	AddObject(key string, marshaler ObjectMarshaler) error
	AddArray(key string, marshaler ArrayMarshaler) error
}

// ArrayEncoder defines a strongly typed, encoding agnostic interface for adding elements to an array.
//
// It provides methods for appending primitive types and nested structures to a
// list. Implementations handle the actual serialization format. This design
// guarantees zero allocations when logging slices or arrays.
type ArrayEncoder interface {
	AppendString(value string)
	AppendInt(value int)
	AppendInt64(value int64)
	AppendBool(value bool)
	AppendFloat64(value float64)
	AppendTime(value time.Time)
	AppendDuration(value time.Duration)
	AppendObject(marshaler ObjectMarshaler) error
	AppendArray(marshaler ArrayMarshaler) error
}

// ObjectMarshaler allows user defined types to efficiently add fields to the logging context.
//
// Implement this interface on your custom structs. The Logger will call
// MarshalLogObject, passing an ObjectEncoder. You then call the encoder's
// methods to add your struct's fields. This avoids reflection and allocation
// overhead during logging.
type ObjectMarshaler interface {
	MarshalLogObject(enc ObjectEncoder) error
}

// ArrayMarshaler allows user defined types to efficiently add elements to the logging context.
//
// Implement this interface on your custom slice or array types. The Logger will
// call MarshalLogArray, passing an ArrayEncoder. You then iterate over your
// collection and call the encoder's methods to append elements. This avoids
// reflection and allocation overhead during logging.
type ArrayMarshaler interface {
	MarshalLogArray(enc ArrayEncoder) error
}
