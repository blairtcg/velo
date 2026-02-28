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

import "sync"

// Buffer is a zero allocation byte buffer pooled for maximum performance.
type buffer struct {
	B []byte
}

var bufPool = sync.Pool{
	New: func() any {
		return &buffer{B: make([]byte, 0, 4096)}
	},
}

func getBuffer() *buffer {
	return bufPool.Get().(*buffer)
}

func (b *buffer) Reset() {
	b.B = b.B[:0]
}

func putBuffer(b *buffer) {
	if cap(b.B) > 64*1024 {
		return
	}
	b.Reset()
	bufPool.Put(b)
}

func (b *buffer) Free() {
	putBuffer(b)
}

func (b *buffer) WriteString(s string) {
	b.B = append(b.B, s...)
}

func (b *buffer) WriteByte(c byte) {
	b.B = append(b.B, c)
}

func (b *buffer) Write(p []byte) (int, error) {
	b.B = append(b.B, p...)
	return len(p), nil
}
