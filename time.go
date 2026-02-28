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

var _smallsString = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

// appendInt appends an integer to a byte slice, zero padded to the specified width.
func appendInt(b []byte, v int, width int) []byte {
	u := uint(v)
	if width == 2 && u < 100 {
		i := u * 2
		return append(b, _smallsString[i], _smallsString[i+1])
	}

	if u == 0 && width <= 1 {
		return append(b, '0')
	}

	// Assemble decimal in reverse order.
	var buf [20]byte
	i := len(buf)
	for u > 0 || width > 0 {
		i--
		q := u / 10
		buf[i] = byte('0' + u - q*10)
		u = q
		width--
	}
	return append(b, buf[i:]...)
}

// appendTime formats a time.Time value and appends it to a byte slice.
//
// It uses a custom, zero allocation encoder for common time formats. This
// optimization significantly outperforms the standard library's
// time.AppendFormat. It falls back to the standard library for unsupported
// formats.
func appendTime(b []byte, t time.Time, format string) []byte {
	switch format {
	case DefaultTimeFormat: // "2006/01/02 15:04:05"
		year, month, day := t.Date()
		hour, min, sec := t.Clock()

		// Year
		y := uint(year)
		if y < 10000 {
			i := (y / 100) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
			i = (y % 100) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
		} else {
			b = appendInt(b, year, 4)
		}

		b = append(b, '/')

		// Month
		i := uint(month) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, '/')

		// Day
		i = uint(day) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, ' ')

		// Hour
		i = uint(hour) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, ':')

		// Min
		i = uint(min) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, ':')

		// Sec
		i = uint(sec) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])

		return b
	case time.RFC3339:
		year, month, day := t.Date()
		hour, min, sec := t.Clock()

		// Year
		y := uint(year)
		if y < 10000 {
			i := (y / 100) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
			i = (y % 100) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
		} else {
			b = appendInt(b, year, 4)
		}

		b = append(b, '-')

		// Month
		i := uint(month) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, '-')

		// Day
		i = uint(day) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, 'T')

		// Hour
		i = uint(hour) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, ':')

		// Min
		i = uint(min) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, ':')

		// Sec
		i = uint(sec) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])

		_, offset := t.Zone()
		if offset == 0 {
			b = append(b, 'Z')
		} else {
			if offset < 0 {
				b = append(b, '-')
				offset = -offset
			} else {
				b = append(b, '+')
			}

			// Offset Hour
			i = uint(offset/3600) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
			b = append(b, ':')

			// Offset Min
			i = uint((offset%3600)/60) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
		}
		return b
	case time.RFC3339Nano:
		year, month, day := t.Date()
		hour, min, sec := t.Clock()

		// Year
		y := uint(year)
		if y < 10000 {
			i := (y / 100) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
			i = (y % 100) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
		} else {
			b = appendInt(b, year, 4)
		}

		b = append(b, '-')

		// Month
		i := uint(month) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, '-')

		// Day
		i = uint(day) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, 'T')

		// Hour
		i = uint(hour) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, ':')

		// Min
		i = uint(min) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])
		b = append(b, ':')

		// Sec
		i = uint(sec) * 2
		b = append(b, _smallsString[i], _smallsString[i+1])

		b = append(b, '.')
		nano := t.Nanosecond()
		b = appendInt(b, nano, 9)

		// Trim trailing zeros for RFC3339Nano
		for len(b) > 0 && b[len(b)-1] == '0' {
			b = b[:len(b)-1]
		}
		if len(b) > 0 && b[len(b)-1] == '.' {
			b = b[:len(b)-1]
		}

		_, offset := t.Zone()
		if offset == 0 {
			b = append(b, 'Z')
		} else {
			if offset < 0 {
				b = append(b, '-')
				offset = -offset
			} else {
				b = append(b, '+')
			}

			// Offset Hour
			i = uint(offset/3600) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
			b = append(b, ':')

			// Offset Min
			i = uint((offset%3600)/60) * 2
			b = append(b, _smallsString[i], _smallsString[i+1])
		}
		return b
	default:
		return t.AppendFormat(b, format)
	}
}
