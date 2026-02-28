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
	"encoding"
	"fmt"
	"os"
	"strconv"
)

var (
	_pid      = os.Getpid()
	_hostname = "unknown"
)

func init() {
	if h, err := os.Hostname(); err == nil {
		_hostname = h
	}
}

// formatAny converts a value to a string efficiently.
//
// It bypasses the reflection heavy fmt.Sprintf for common types, significantly
// improving performance during log formatting.
func formatAny(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case bool:
		return strconv.FormatBool(val)
	case error:
		return val.Error()
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case fmt.Stringer:
		return val.String()
	case encoding.TextMarshaler:
		if b, err := val.MarshalText(); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%+v", val)
	default:
		return fmt.Sprintf("%+v", val)
	}
}
