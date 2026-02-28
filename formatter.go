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
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var _defaultStyles = DefaultStyles()

// formatLogText formats a log entry directly onto a pooled buffer.
//
// It bypasses the Entry struct allocation, providing maximum performance for
// simple text logs.
func formatLogText(b *buffer, l *Logger, cfg *loggerConfig, level Level, msg string, callFields []any, callTypedFields []Field, ctxFields []Field, t time.Time) {
	st := _defaultStyles

	// timestamp
	if !t.IsZero() {
		var buf [64]byte
		tb := appendTime(buf[:0], t, cfg.timeFormat)
		b.WriteString(st.Timestamp.Render(string(tb)))
		b.WriteByte(' ')
	}

	// level
	if level != noLevel {
		if s, ok := st.CachedLevelStrings[level]; ok {
			b.WriteString(s)
			b.WriteByte(' ')
		} else if lvlStyle, ok := st.Levels[level]; ok {
			b.WriteString(lvlStyle.String())
			b.WriteByte(' ')
		}
	}

	// prefix
	if cfg.prefix != "" {
		b.WriteString(st.Prefix.Render(cfg.prefix + ":"))
		b.WriteByte(' ')
	}

	// message
	if msg != "" {
		b.WriteString(st.Message.Render(msg))
	}

	// Helper to process fields
	processFields := func(fields []any) {
		for i := 0; i < len(fields); i += 2 {
			if i+1 >= len(fields) {
				break
			}

			key := formatAny(fields[i])
			val := formatAny(fields[i+1])

			if key == "" {
				continue
			}

			b.WriteByte(' ')

			keyStr := st.Key.Render(key)
			if ks, ok := st.Keys[key]; ok {
				keyStr = ks.Render(key)
			}

			valStr := st.Value.Render(val)
			if vs, ok := st.Values[key]; ok {
				valStr = vs.Render(val)
			}

			sep := st.Separator.Render("=")

			b.WriteString(keyStr)
			b.WriteString(sep)
			if strings.Contains(val, " ") || strings.Contains(val, "=") {
				b.WriteString(`"` + valStr + `"`)
			} else {
				b.WriteString(valStr)
			}
		}
	}

	processFields(l.fields)
	processFields(callFields)

	// Helper to process typed fields
	processTypedFields := func(fields []Field) {
		for i := 0; i < len(fields); i++ {
			f := &fields[i]
			if f.Key == "" {
				continue
			}

			b.WriteByte(' ')

			keyStr := st.Key.Render(f.Key)
			if ks, ok := st.Keys[f.Key]; ok {
				keyStr = ks.Render(f.Key)
			}

			var val string
			switch f.Type {
			case StringType:
				val = f.Str
			case IntType:
				val = strconv.FormatInt(f.Int, 10)
			case BoolType:
				val = strconv.FormatBool(f.Int == 1)
			case ErrorType:
				if f.Any != nil {
					val = f.Any.(error).Error()
				}
			case TimeType:
				var buf [64]byte
				tb := appendTime(buf[:0], time.Unix(0, f.Int), cfg.timeFormat)
				val = string(tb)
			case DurationType:
				val = time.Duration(f.Int).String()
			case ObjectType:
				var buf buffer
				sub := getJSONEncoder(&buf)
				buf.WriteByte('{')
				if f.Any != nil {
					f.Any.(ObjectMarshaler).MarshalLogObject(sub)
				}
				buf.WriteByte('}')
				putJSONEncoder(sub)
				val = string(buf.B)
			case ArrayType:
				var buf buffer
				sub := getJSONEncoder(&buf)
				buf.WriteByte('[')
				if f.Any != nil {
					f.Any.(ArrayMarshaler).MarshalLogArray(sub)
				}
				buf.WriteByte(']')
				putJSONEncoder(sub)
				val = string(buf.B)
			case IntsType:
				var buf buffer
				buf.WriteByte('[')
				if f.Int > 0 {
					slice := unsafe.Slice((*int)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
					for i, v := range slice {
						if i > 0 {
							buf.WriteByte(',')
						}
						buf.B = strconv.AppendInt(buf.B, int64(v), 10)
					}
				}
				buf.WriteByte(']')
				val = string(buf.B)
			case StringsType:
				var buf buffer
				buf.WriteByte('[')
				if f.Int > 0 {
					slice := unsafe.Slice((*string)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
					for i, v := range slice {
						if i > 0 {
							buf.WriteByte(',')
						}
						appendJSONString(&buf, v)
					}
				}
				buf.WriteByte(']')
				val = string(buf.B)
			case TimesType:
				var buf buffer
				buf.WriteByte('[')
				if f.Int > 0 {
					slice := unsafe.Slice((*time.Time)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
					for i, v := range slice {
						if i > 0 {
							buf.WriteByte(',')
						}
						buf.WriteByte('"')
						buf.B = appendTime(buf.B, v, cfg.timeFormat)
						buf.WriteByte('"')
					}
				}
				buf.WriteByte(']')
				val = string(buf.B)
			case AnyType:
				val = formatAny(f.Any)
			}

			valStr := st.Value.Render(val)
			if vs, ok := st.Values[f.Key]; ok {
				valStr = vs.Render(val)
			}

			sep := st.Separator.Render("=")

			b.WriteString(keyStr)
			b.WriteString(sep)
			if strings.Contains(val, " ") || strings.Contains(val, "=") {
				b.WriteString(`"` + valStr + `"`)
			} else {
				b.WriteString(valStr)
			}
		}
	}

	processTypedFields(l.typedFields)
	processTypedFields(ctxFields)
	processTypedFields(callTypedFields)

	b.WriteByte('\n')
}

// formatLogJSON formats a log entry directly onto a pooled buffer as JSON.
//
// It bypasses the Entry struct allocation, providing maximum performance for
// simple JSON logs.
func formatLogJSON(b *buffer, l *Logger, cfg *loggerConfig, level Level, msg string, callFields []any, callTypedFields []Field, ctxFields []Field, t time.Time) {
	first := true

	if !t.IsZero() {
		b.B = append(b.B, '{', '"', 't', 'i', 'm', 'e', '"', ':')
		switch cfg.timeFormat {
		case "unix":
			b.B = strconv.AppendInt(b.B, t.Unix(), 10)
		case "unix_milli":
			b.B = strconv.AppendInt(b.B, t.UnixMilli(), 10)
		default:
			b.B = append(b.B, '"')
			b.B = appendTime(b.B, t, cfg.timeFormat)
			b.B = append(b.B, '"')
		}
		first = false
	} else {
		b.B = append(b.B, '{')
	}

	if level != noLevel {
		if !first {
			b.B = append(b.B, ',')
		}
		first = false
		b.B = append(b.B, level.JSONField()...)
	}

	if cfg.prefix != "" {
		if !first {
			b.B = append(b.B, ',', '"', 'p', 'r', 'e', 'f', 'i', 'x', '"', ':')
		} else {
			b.B = append(b.B, '"', 'p', 'r', 'e', 'f', 'i', 'x', '"', ':')
		}
		first = false
		appendJSONString(b, cfg.prefix)
	}

	if msg != "" {
		if !first {
			b.B = append(b.B, ',', '"', 'm', 's', 'g', '"', ':')
		} else {
			b.B = append(b.B, '"', 'm', 's', 'g', '"', ':')
		}
		first = false
		appendJSONString(b, msg)
	}

	// pre-encoded json fields
	preEncoded := l.preEncodedJSON
	hasPreEncoded := len(preEncoded) > 0 || (len(l.fields) == 0 && len(l.typedFields) == 0)
	if hasPreEncoded && len(preEncoded) > 0 {
		if first {
			// Skip leading comma if this is the first item
			b.B = append(b.B, preEncoded[1:]...)
		} else {
			b.B = append(b.B, preEncoded...)
		}
		first = false
	}

	// logger fields (if not pre-encoded)
	if !hasPreEncoded {
		for i := 0; i < len(l.fields); i += 2 {
			if i+1 < len(l.fields) {
				encodeKeyValToJSON(b, l.fields[i], l.fields[i+1], !first)
				first = false
			}
		}
		for i := 0; i < len(l.typedFields); i++ {
			encodeFieldToJSON(b, &l.typedFields[i], cfg.timeFormat, !first)
			first = false
		}
	}

	for i := 0; i < len(callFields); i += 2 {
		if i+1 < len(callFields) {
			encodeKeyValToJSON(b, callFields[i], callFields[i+1], !first)
			first = false
		}
	}

	for i := 0; i < len(ctxFields); i++ {
		encodeFieldToJSON(b, &ctxFields[i], cfg.timeFormat, !first)
		first = false
	}

	for i := 0; i < len(callTypedFields); i++ {
		encodeFieldToJSON(b, &callTypedFields[i], cfg.timeFormat, !first)
		first = false
	}

	b.B = append(b.B, '}', '\n')
}

// formatEntry formats a log entry into a string or JSON directly onto a pooled buffer.
func formatEntry(b *buffer, e *Entry) {
	switch e.Formatter {
	case JSONFormatter:
		formatJSON(b, e)
	case TextFormatter:
		fallthrough
	default:
		formatText(b, e)
	}
}

func formatText(b *buffer, e *Entry) {
	st := _defaultStyles

	// timestamp
	if !e.Time.IsZero() {
		// appendTime is a custom zero allocation encoder for common formats
		var buf [64]byte
		tb := appendTime(buf[:0], e.Time, e.TimeFormat)
		b.WriteString(st.Timestamp.Render(string(tb)))
		b.WriteByte(' ')
	}

	// level
	if e.Level != noLevel {
		if lvlStyle, ok := st.Levels[e.Level]; ok {
			b.WriteString(lvlStyle.String())
			b.WriteByte(' ')
		}
	}

	// caller
	if e.Caller != "" {
		caller := "<" + e.Caller + ">"
		b.WriteString(st.Caller.Render(caller))
		b.WriteByte(' ')
	}

	// prefix
	if e.Prefix != "" {
		b.WriteString(st.Prefix.Render(e.Prefix + ":"))
		b.WriteByte(' ')
	}

	// message
	if e.Message != "" {
		b.WriteString(st.Message.Render(e.Message))
	}

	// fields
	for i := 0; i < len(e.Fields); i += 2 {
		if i+1 >= len(e.Fields) {
			break
		}

		key := formatAny(e.Fields[i])
		val := formatAny(e.Fields[i+1])

		if key == "" {
			continue
		}

		b.WriteByte(' ')

		keyStr := st.Key.Render(key)
		if ks, ok := st.Keys[key]; ok {
			keyStr = ks.Render(key)
		}

		valStr := st.Value.Render(val)
		if vs, ok := st.Values[key]; ok {
			valStr = vs.Render(val)
		}

		sep := st.Separator.Render("=")

		b.WriteString(keyStr)
		b.WriteString(sep)
		if strings.Contains(val, " ") || strings.Contains(val, "=") {
			b.WriteString(`"` + valStr + `"`)
		} else {
			b.WriteString(valStr)
		}
	}

	// typed fields
	for i := 0; i < len(e.TypedFields); i++ {
		f := &e.TypedFields[i]
		if f.Key == "" {
			continue
		}

		b.WriteByte(' ')

		keyStr := st.Key.Render(f.Key)
		if ks, ok := st.Keys[f.Key]; ok {
			keyStr = ks.Render(f.Key)
		}

		var val string
		switch f.Type {
		case StringType:
			val = f.Str
		case IntType:
			val = strconv.FormatInt(f.Int, 10)
		case BoolType:
			val = strconv.FormatBool(f.Int == 1)
		case ErrorType:
			if f.Any != nil {
				val = f.Any.(error).Error()
			}
		case TimeType:
			var buf [64]byte
			tb := appendTime(buf[:0], time.Unix(0, f.Int), e.TimeFormat)
			val = string(tb)
		case DurationType:
			val = time.Duration(f.Int).String()
		case ObjectType:
			// For text format, we can just use JSON encoding for the object
			var buf buffer
			sub := getJSONEncoder(&buf)
			buf.WriteByte('{')
			if f.Any != nil {
				f.Any.(ObjectMarshaler).MarshalLogObject(sub)
			}
			buf.WriteByte('}')
			putJSONEncoder(sub)
			val = string(buf.B)
		case ArrayType:
			var buf buffer
			sub := getJSONEncoder(&buf)
			buf.WriteByte('[')
			if f.Any != nil {
				f.Any.(ArrayMarshaler).MarshalLogArray(sub)
			}
			buf.WriteByte(']')
			putJSONEncoder(sub)
			val = string(buf.B)
		case IntsType:
			var buf buffer
			buf.WriteByte('[')
			if f.Int > 0 {
				slice := unsafe.Slice((*int)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
				for i, v := range slice {
					if i > 0 {
						buf.WriteByte(',')
					}
					buf.B = strconv.AppendInt(buf.B, int64(v), 10)
				}
			}
			buf.WriteByte(']')
			val = string(buf.B)
		case StringsType:
			var buf buffer
			buf.WriteByte('[')
			if f.Int > 0 {
				slice := unsafe.Slice((*string)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
				for i, v := range slice {
					if i > 0 {
						buf.WriteByte(',')
					}
					appendJSONString(&buf, v)
				}
			}
			buf.WriteByte(']')
			val = string(buf.B)
		case TimesType:
			var buf buffer
			buf.WriteByte('[')
			if f.Int > 0 {
				slice := unsafe.Slice((*time.Time)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
				for i, v := range slice {
					if i > 0 {
						buf.WriteByte(',')
					}
					buf.WriteByte('"')
					buf.B = appendTime(buf.B, v, e.TimeFormat)
					buf.WriteByte('"')
				}
			}
			buf.WriteByte(']')
			val = string(buf.B)
		case AnyType:
			val = formatAny(f.Any)
		}

		valStr := st.Value.Render(val)
		if vs, ok := st.Values[f.Key]; ok {
			valStr = vs.Render(val)
		}

		sep := st.Separator.Render("=")

		b.WriteString(keyStr)
		b.WriteString(sep)
		if strings.Contains(val, " ") || strings.Contains(val, "=") {
			b.WriteString(`"` + valStr + `"`)
		} else {
			b.WriteString(valStr)
		}
	}

	if len(e.Stack) > 0 {
		b.WriteByte('\n')
		writeStacktrace(b, e.Stack, st)
		// strip trailing newline from buf to avoid double newline since formatText adds one
		if len(b.B) > 0 && b.B[len(b.B)-1] == '\n' {
			b.B = b.B[:len(b.B)-1]
		}
	}

	b.WriteByte('\n')
}

// formatJSON provides a custom, zero allocation JSON encoder.
//
// It completely bypasses the standard library's json.Marshal. This eliminates
// map allocations and reflection, significantly improving serialization speed.
func formatJSON(b *buffer, e *Entry) {
	first := true
	addSep := func() {
		if !first {
			b.B = append(b.B, ',')
		}
		first = false
	}

	if !e.Time.IsZero() {
		b.B = append(b.B, '{', '"', 't', 'i', 'm', 'e', '"', ':')
		switch e.TimeFormat {
		case "unix":
			b.B = strconv.AppendInt(b.B, e.Time.Unix(), 10)
		case "unix_milli":
			b.B = strconv.AppendInt(b.B, e.Time.UnixMilli(), 10)
		default:
			b.B = append(b.B, '"')
			b.B = appendTime(b.B, e.Time, e.TimeFormat)
			b.B = append(b.B, '"')
		}
		first = false
	} else {
		b.B = append(b.B, '{')
	}

	if e.Level != noLevel {
		addSep()
		b.B = append(b.B, e.Level.JSONField()...)
	}

	if e.Caller != "" {
		if !first {
			b.B = append(b.B, ',', '"', 'c', 'a', 'l', 'l', 'e', 'r', '"', ':')
		} else {
			b.B = append(b.B, '"', 'c', 'a', 'l', 'l', 'e', 'r', '"', ':')
		}
		first = false
		appendJSONString(b, e.Caller)
	}

	if e.Prefix != "" {
		if !first {
			b.B = append(b.B, ',', '"', 'p', 'r', 'e', 'f', 'i', 'x', '"', ':')
		} else {
			b.B = append(b.B, '"', 'p', 'r', 'e', 'f', 'i', 'x', '"', ':')
		}
		first = false
		appendJSONString(b, e.Prefix)
	}

	if e.Message != "" {
		if !first {
			b.B = append(b.B, ',', '"', 'm', 's', 'g', '"', ':')
		} else {
			b.B = append(b.B, '"', 'm', 's', 'g', '"', ':')
		}
		first = false
		appendJSONString(b, e.Message)
	}

	// pre-encoded json fields
	if len(e.PreEncodedJSON) > 0 {
		if first {
			// Skip leading comma if this is the first item
			b.B = append(b.B, e.PreEncodedJSON[1:]...)
		} else {
			b.B = append(b.B, e.PreEncodedJSON...)
		}
		first = false
	}

	// fields
	for i := 0; i < len(e.Fields); i += 2 {
		if i+1 < len(e.Fields) {
			encodeKeyValToJSON(b, e.Fields[i], e.Fields[i+1], !first)
			first = false
		}
	}

	// typed fields
	for i := 0; i < len(e.TypedFields); i++ {
		encodeFieldToJSON(b, &e.TypedFields[i], e.TimeFormat, !first)
		first = false
	}

	b.B = append(b.B, '}', '\n')
}

// encodeKeyValToJSON encodes a loosely typed key-value pair to JSON.
func encodeKeyValToJSON(b *buffer, key, val any, prependComma bool) {
	// Optimize for string keys to avoid formatAny call
	if k, ok := key.(string); ok {
		appendJSONKey(b, k, prependComma)
	} else {
		appendJSONKey(b, formatAny(key), prependComma)
	}
	appendJSONAny(b, val)
}

// encodeFieldToJSON encodes a strongly typed Field to JSON and appends it to the buffer.
func encodeFieldToJSON(b *buffer, f *Field, timeFormat string, prependComma bool) {
	appendJSONKey(b, f.Key, prependComma)
	switch f.Type {
	case StringType:
		appendJSONString(b, f.Str)
	case IntType:
		b.B = strconv.AppendInt(b.B, f.Int, 10)
	case BoolType:
		b.B = strconv.AppendBool(b.B, f.Int == 1)
	case ErrorType:
		if f.Any != nil {
			appendJSONString(b, f.Any.(error).Error())
		} else {
			b.B = append(b.B, "null"...)
		}
	case TimeType:
		b.B = append(b.B, '"')
		b.B = appendTime(b.B, time.Unix(0, f.Int), timeFormat)
		b.B = append(b.B, '"')
	case DurationType:
		b.B = strconv.AppendInt(b.B, f.Int, 10)
	case ObjectType:
		b.B = append(b.B, '{')
		sub := getJSONEncoder(b)
		if f.Any != nil {
			f.Any.(ObjectMarshaler).MarshalLogObject(sub)
		}
		putJSONEncoder(sub)
		b.B = append(b.B, '}')
	case ArrayType:
		b.B = append(b.B, '[')
		sub := getJSONEncoder(b)
		if f.Any != nil {
			f.Any.(ArrayMarshaler).MarshalLogArray(sub)
		}
		putJSONEncoder(sub)
		b.B = append(b.B, ']')
	case IntsType:
		b.B = append(b.B, '[')
		if f.Int > 0 {
			slice := unsafe.Slice((*int)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
			for i, v := range slice {
				if i > 0 {
					b.B = append(b.B, ',')
				}
				b.B = strconv.AppendInt(b.B, int64(v), 10)
			}
		}
		b.B = append(b.B, ']')
	case StringsType:
		b.B = append(b.B, '[')
		if f.Int > 0 {
			slice := unsafe.Slice((*string)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
			for i, v := range slice {
				if i > 0 {
					b.B = append(b.B, ',')
				}
				appendJSONString(b, v)
			}
		}
		b.B = append(b.B, ']')
	case TimesType:
		b.B = append(b.B, '[')
		if f.Int > 0 {
			slice := unsafe.Slice((*time.Time)(unsafe.Pointer(unsafe.StringData(f.Str))), int(f.Int))
			for i, v := range slice {
				if i > 0 {
					b.B = append(b.B, ',')
				}
				b.B = append(b.B, '"')
				b.B = appendTime(b.B, v, timeFormat)
				b.B = append(b.B, '"')
			}
		}
		b.B = append(b.B, ']')
	case AnyType:
		appendJSONAny(b, f.Any)
	}
}

var _noEscape [256]bool

func init() {
	for i := 0; i <= 0x1f; i++ {
		_noEscape[i] = true
	}
	_noEscape['"'] = true
	_noEscape['\\'] = true
}

var _hex = "0123456789abcdef"

// appendJSONKey appends a JSON key to the buffer without allocating memory.
func appendJSONKey(b *buffer, s string, prependComma bool) {
	if prependComma {
		b.B = append(b.B, ',', '"')
	} else {
		b.B = append(b.B, '"')
	}
	for i := 0; i < len(s); i++ {
		if _noEscape[s[i]] {
			b.B = append(b.B, s[:i]...)
			appendJSONStringEscape(b, s, i)
			b.B = append(b.B, '"', ':')
			return
		}
	}
	b.B = append(b.B, s...)
	b.B = append(b.B, '"', ':')
}

// appendJSONString appends a properly escaped JSON string to the buffer without allocating memory.
//
// It uses chunked memory copies for maximum performance, mirroring Zap's safeSet approach.
func appendJSONString(b *buffer, s string) {
	b.B = append(b.B, '"')
	for i := 0; i < len(s); i++ {
		if _noEscape[s[i]] {
			b.B = append(b.B, s[:i]...)
			appendJSONStringEscape(b, s, i)
			b.B = append(b.B, '"')
			return
		}
	}
	b.B = append(b.B, s...)
	b.B = append(b.B, '"')
}

func appendJSONStringEscape(b *buffer, s string, i int) {
	start := i
	for ; i < len(s); i++ {
		if _noEscape[s[i]] {
			if start < i {
				b.B = append(b.B, s[start:i]...)
			}
			c := s[i]
			switch c {
			case '"':
				b.B = append(b.B, '\\', '"')
			case '\\':
				b.B = append(b.B, '\\', '\\')
			case '\n':
				b.B = append(b.B, '\\', 'n')
			case '\r':
				b.B = append(b.B, '\\', 'r')
			case '\t':
				b.B = append(b.B, '\\', 't')
			case '\b':
				b.B = append(b.B, '\\', 'b')
			case '\f':
				b.B = append(b.B, '\\', 'f')
			default:
				b.B = append(b.B, '\\', 'u', '0', '0', _hex[c>>4], _hex[c&0xF])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		b.B = append(b.B, s[start:]...)
	}
}

// appendJSONAny appends an arbitrary value to the buffer as json without allocating for common types.
func appendJSONAny(b *buffer, v any) {
	switch val := v.(type) {
	case string:
		appendJSONString(b, val)
	case int:
		b.B = strconv.AppendInt(b.B, int64(val), 10)
	case int64:
		b.B = strconv.AppendInt(b.B, val, 10)
	case bool:
		b.B = strconv.AppendBool(b.B, val)
	case ObjectMarshaler:
		b.B = append(b.B, '{')
		enc := getJSONEncoder(b)
		val.MarshalLogObject(enc)
		putJSONEncoder(enc)
		b.B = append(b.B, '}')
	case ArrayMarshaler:
		b.B = append(b.B, '[')
		enc := getJSONEncoder(b)
		val.MarshalLogArray(enc)
		putJSONEncoder(enc)
		b.B = append(b.B, ']')
	case error:
		appendJSONString(b, val.Error())
	case time.Time:
		b.B = append(b.B, '"')
		b.B = appendTime(b.B, val, time.RFC3339Nano)
		b.B = append(b.B, '"')
	case int32:
		b.B = strconv.AppendInt(b.B, int64(val), 10)
	case uint:
		b.B = strconv.AppendUint(b.B, uint64(val), 10)
	case uint64:
		b.B = strconv.AppendUint(b.B, val, 10)
	case uint32:
		b.B = strconv.AppendUint(b.B, uint64(val), 10)
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			b.B = append(b.B, "null"...)
		} else {
			b.B = strconv.AppendFloat(b.B, val, 'f', -1, 64)
		}
	case float32:
		if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
			b.B = append(b.B, "null"...)
		} else {
			b.B = strconv.AppendFloat(b.B, float64(val), 'f', -1, 32)
		}
	case []int:
		b.B = append(b.B, '[')
		for i, v := range val {
			if i > 0 {
				b.B = append(b.B, ',')
			}
			b.B = strconv.AppendInt(b.B, int64(v), 10)
		}
		b.B = append(b.B, ']')
	case []string:
		b.B = append(b.B, '[')
		for i, v := range val {
			if i > 0 {
				b.B = append(b.B, ',')
			}
			appendJSONString(b, v)
		}
		b.B = append(b.B, ']')
	case []any:
		b.B = append(b.B, '[')
		for i, v := range val {
			if i > 0 {
				b.B = append(b.B, ',')
			}
			appendJSONAny(b, v)
		}
		b.B = append(b.B, ']')
	case map[string]any:
		b.B = append(b.B, '{')
		first := true
		for k, v := range val {
			if !first {
				b.B = append(b.B, ',')
			}
			first = false
			appendJSONString(b, k)
			b.B = append(b.B, ':')
			appendJSONAny(b, v)
		}
		b.B = append(b.B, '}')
	case []time.Time:
		b.B = append(b.B, '[')
		for i, v := range val {
			if i > 0 {
				b.B = append(b.B, ',')
			}
			b.B = append(b.B, '"')
			b.B = appendTime(b.B, v, time.RFC3339Nano)
			b.B = append(b.B, '"')
		}
		b.B = append(b.B, ']')
	case int8:
		b.B = strconv.AppendInt(b.B, int64(val), 10)
	case int16:
		b.B = strconv.AppendInt(b.B, int64(val), 10)
	case uint8:
		b.B = strconv.AppendUint(b.B, uint64(val), 10)
	case uint16:
		b.B = strconv.AppendUint(b.B, uint64(val), 10)
	case []byte:
		appendJSONString(b, string(val))
	case time.Duration:
		b.B = strconv.AppendInt(b.B, int64(val), 10)
	case fmt.Stringer:
		appendJSONString(b, val.String())
	case encoding.TextMarshaler:
		if txt, err := val.MarshalText(); err == nil {
			appendJSONString(b, string(txt))
		} else {
			appendJSONString(b, err.Error())
		}
	case json.Marshaler:
		if buf, err := val.MarshalJSON(); err == nil {
			b.Write(buf)
		} else {
			appendJSONString(b, err.Error())
		}
	default:
		appendJSONString(b, formatAny(val))
	}
}
