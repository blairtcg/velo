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
	"strconv"
	"sync"
	"time"
)

// JSONEncoder provides a low allocation JSON encoder.
//
// It implements both the ObjectEncoder and ArrayEncoder interfaces. The Logger
// uses this internally to serialize complex, user defined types without relying
// on the standard library's reflection heavy json package.
type JSONEncoder struct {
	buf   *buffer
	first bool
}

var _jsonEncoderPool = sync.Pool{
	New: func() interface{} {
		return &JSONEncoder{}
	},
}

func getJSONEncoder(b *buffer) *JSONEncoder {
	enc := _jsonEncoderPool.Get().(*JSONEncoder)
	enc.buf = b
	enc.first = true
	return enc
}

func putJSONEncoder(enc *JSONEncoder) {
	enc.buf = nil
	_jsonEncoderPool.Put(enc)
}

func (enc *JSONEncoder) addKey(key string) {
	appendJSONKey(enc.buf, key, !enc.first)
	enc.first = false
}

func (enc *JSONEncoder) addSep() {
	if !enc.first {
		enc.buf.WriteByte(',')
	}
	enc.first = false
}

// ObjectEncoder implementation
func (enc *JSONEncoder) AddString(key, value string) {
	enc.addKey(key)
	appendJSONString(enc.buf, value)
}

func (enc *JSONEncoder) AddInt(key string, value int) {
	enc.addKey(key)
	enc.buf.B = strconv.AppendInt(enc.buf.B, int64(value), 10)
}

func (enc *JSONEncoder) AddInt64(key string, value int64) {
	enc.addKey(key)
	enc.buf.B = strconv.AppendInt(enc.buf.B, value, 10)
}

func (enc *JSONEncoder) AddBool(key string, value bool) {
	enc.addKey(key)
	enc.buf.B = strconv.AppendBool(enc.buf.B, value)
}

func (enc *JSONEncoder) AddFloat64(key string, value float64) {
	enc.addKey(key)
	enc.buf.B = strconv.AppendFloat(enc.buf.B, value, 'f', -1, 64)
}

func (enc *JSONEncoder) AddTime(key string, value time.Time) {
	enc.addKey(key)
	enc.buf.WriteByte('"')
	enc.buf.B = appendTime(enc.buf.B, value, time.RFC3339Nano)
	enc.buf.WriteByte('"')
}

func (enc *JSONEncoder) AddDuration(key string, value time.Duration) {
	enc.addKey(key)
	enc.buf.B = strconv.AppendInt(enc.buf.B, value.Nanoseconds(), 10)
}

func (enc *JSONEncoder) AddObject(key string, marshaler ObjectMarshaler) error {
	enc.addKey(key)
	enc.buf.WriteByte('{')
	if marshaler != nil {
		enc.first = true
		marshaler.MarshalLogObject(enc)
	}
	enc.first = false
	enc.buf.WriteByte('}')
	return nil
}

func (enc *JSONEncoder) AddArray(key string, marshaler ArrayMarshaler) error {
	enc.addKey(key)
	enc.buf.WriteByte('[')
	if marshaler != nil {
		enc.first = true
		marshaler.MarshalLogArray(enc)
	}
	enc.first = false
	enc.buf.WriteByte(']')
	return nil
}

// ArrayEncoder implementation
func (enc *JSONEncoder) AppendString(value string) {
	enc.addSep()
	appendJSONString(enc.buf, value)
}

func (enc *JSONEncoder) AppendInt(value int) {
	enc.addSep()
	enc.buf.B = strconv.AppendInt(enc.buf.B, int64(value), 10)
}

func (enc *JSONEncoder) AppendInt64(value int64) {
	enc.addSep()
	enc.buf.B = strconv.AppendInt(enc.buf.B, value, 10)
}

func (enc *JSONEncoder) AppendBool(value bool) {
	enc.addSep()
	enc.buf.B = strconv.AppendBool(enc.buf.B, value)
}

func (enc *JSONEncoder) AppendFloat64(value float64) {
	enc.addSep()
	enc.buf.B = strconv.AppendFloat(enc.buf.B, value, 'f', -1, 64)
}

func (enc *JSONEncoder) AppendTime(value time.Time) {
	enc.addSep()
	enc.buf.WriteByte('"')
	enc.buf.B = appendTime(enc.buf.B, value, time.RFC3339Nano)
	enc.buf.WriteByte('"')
}

func (enc *JSONEncoder) AppendDuration(value time.Duration) {
	enc.addSep()
	enc.buf.B = strconv.AppendInt(enc.buf.B, value.Nanoseconds(), 10)
}

func (enc *JSONEncoder) AppendObject(marshaler ObjectMarshaler) error {
	enc.addSep()
	enc.buf.WriteByte('{')
	if marshaler != nil {
		enc.first = true
		marshaler.MarshalLogObject(enc)
	}
	enc.first = false
	enc.buf.WriteByte('}')
	return nil
}

func (enc *JSONEncoder) AppendArray(marshaler ArrayMarshaler) error {
	enc.addSep()
	enc.buf.WriteByte('[')
	if marshaler != nil {
		enc.first = true
		marshaler.MarshalLogArray(enc)
	}
	enc.first = false
	enc.buf.WriteByte(']')
	return nil
}
