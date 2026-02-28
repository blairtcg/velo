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
	"runtime"
	"strconv"
	"strings"
)

const maxTraceDepth = 5

// writeStacktrace processes program counters into a human readable, styled stack trace.
//
// It avoids string splitting and regular expressions, relying entirely on
// runtime.CallersFrames. This approach ensures high performance, comparable to
// Zap's stack trace generation.
//
//go:noinline
func writeStacktrace(b *buffer, pcs []uintptr, st *Styles) {
	if len(pcs) == 0 {
		return
	}

	frames := runtime.CallersFrames(pcs)
	rendered := 0

	// cache static byte slices to eliminate loop allocations.
	prefix := []byte(st.Separator.Render("   at "))
	space := byte(' ')
	newline := byte('\n')

	for {
		frame, more := frames.Next()

		// ignore standard library internals and test runners.
		if strings.Contains(frame.File, "runtime/") || strings.Contains(frame.File, "testing/") {
			if !more {
				break
			}
			continue
		}

		// ignore our own library frames unless we are running tests.
		if strings.Contains(frame.Function, "velo") && !strings.HasSuffix(frame.File, "_test.go") {
			if !more {
				break
			}
			continue
		}

		if rendered >= maxTraceDepth {
			break
		}

		// isolate the function name from its package path.
		fn := frame.Function
		if idx := strings.LastIndexByte(fn, '/'); idx >= 0 {
			fn = fn[idx+1:]
		}
		if idx := strings.IndexByte(fn, '.'); idx >= 0 {
			fn = fn[idx+1:]
		}

		// isolate the file name from its absolute path.
		file := frame.File
		if idx := strings.LastIndexByte(file, '/'); idx >= 0 {
			file = file[idx+1:]
		}

		// stream the styled output directly to the buffer.
		b.Write(prefix)
		b.WriteString(st.StackFunc.Render(fn))
		b.WriteByte(space)

		// concatenate file and line efficiently.
		loc := file + ":" + strconv.Itoa(frame.Line)
		b.WriteString(st.StackFile.Render(loc))
		b.WriteByte(newline)

		rendered++
		if !more {
			break
		}
	}
}
