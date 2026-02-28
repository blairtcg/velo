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
	"time"
	"unsafe"
)

// FieldType identifies the underlying data type of a strongly typed Field.
type FieldType uint8

const (
	// StringType indicates a string field.
	StringType FieldType = iota
	// IntType indicates an integer field.
	IntType
	// BoolType indicates a boolean field.
	BoolType
	// ErrorType indicates an error field.
	ErrorType
	// TimeType indicates a time.Time field.
	TimeType
	// DurationType indicates a time.Duration field.
	DurationType
	// AnyType indicates an arbitrary interface{} field.
	AnyType
	// ObjectType indicates a field implementing ObjectMarshaler.
	ObjectType
	// ArrayType indicates a field implementing ArrayMarshaler.
	ArrayType
	// IntsType indicates a slice of integers.
	IntsType
	// StringsType indicates a slice of strings.
	StringsType
	// TimesType indicates a slice of time.Time values.
	TimesType
)

// Field represents a strongly typed key-value pair.
//
// It avoids the interface{} boxing overhead associated with standard variadic
// arguments. Using Fields enables true zero allocation logging on the hot path.
type Field struct {
	Key  string
	Str  string
	Any  any
	Int  int64
	Type FieldType
}

// String constructs a Field containing a string value.
func String(key, val string) Field { return Field{Key: key, Type: StringType, Str: val} }

// Int constructs a Field containing an integer value.
func Int(key string, val int) Field { return Field{Key: key, Type: IntType, Int: int64(val)} }

// Int64 constructs a Field containing a 64-bit integer value.
func Int64(key string, val int64) Field { return Field{Key: key, Type: IntType, Int: val} }

// Bool constructs a Field containing a boolean value.
func Bool(key string, val bool) Field {
	var i int64
	if val {
		i = 1
	}
	return Field{Key: key, Type: BoolType, Int: i}
}

// Time constructs a Field containing a time.Time value.
func Time(key string, val time.Time) Field {
	return Field{Key: key, Type: TimeType, Int: val.UnixNano()}
}

// Duration constructs a Field containing a time.Duration value.
func Duration(key string, val time.Duration) Field {
	return Field{Key: key, Type: DurationType, Int: int64(val)}
}

// Err constructs a Field containing an error value.
//
// It automatically uses the key "error".
func Err(err error) Field { return Field{Key: "error", Type: ErrorType, Any: err} }

// Any constructs a Field containing an arbitrary interface{} value.
//
// Performance Note: Using Any incurs allocation overhead due to interface boxing.
// Prefer strongly typed constructors (like String or Int) when possible.
func Any(key string, val any) Field { return Field{Key: key, Type: AnyType, Any: val} }

// Object constructs a Field containing an ObjectMarshaler.
//
// Use this to log complex structs with zero allocations.
func Object(key string, val ObjectMarshaler) Field {
	return Field{Key: key, Type: ObjectType, Any: val}
}

// Array constructs a Field containing an ArrayMarshaler.
//
// Use this to log collections with zero allocations.
func Array(key string, val ArrayMarshaler) Field { return Field{Key: key, Type: ArrayType, Any: val} }

// Ints constructs a Field containing a slice of integers.
func Ints(key string, val []int) Field {
	if len(val) == 0 {
		return Field{Key: key, Type: IntsType, Int: 0}
	}
	return Field{Key: key, Type: IntsType, Str: unsafe.String((*byte)(unsafe.Pointer(&val[0])), 1), Int: int64(len(val))}
}

// Strings constructs a Field containing a slice of strings.
func Strings(key string, val []string) Field {
	if len(val) == 0 {
		return Field{Key: key, Type: StringsType, Int: 0}
	}
	return Field{Key: key, Type: StringsType, Str: unsafe.String((*byte)(unsafe.Pointer(&val[0])), 1), Int: int64(len(val))}
}

// Times constructs a Field containing a slice of time.Time values.
func Times(key string, val []time.Time) Field {
	if len(val) == 0 {
		return Field{Key: key, Type: TimesType, Int: 0}
	}
	return Field{Key: key, Type: TimesType, Str: unsafe.String((*byte)(unsafe.Pointer(&val[0])), 1), Int: int64(len(val))}
}
