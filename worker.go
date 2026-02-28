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
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

var (
	_workers   []*worker
	_workersMu sync.Mutex
)

func flushAllWorkers() {
	_workersMu.Lock()
	defer _workersMu.Unlock()
	for _, w := range _workers {
		w.sync()
	}
}

// worker manages a background goroutine that consumes log entries from a queue.
//
// It formats the entries and writes them to the configured output. This struct
// forms the core of the asynchronous logging system, ensuring the main application
// thread is not blocked by I/O operations.
type worker struct {
	queue    chan *buffer
	syncChan chan chan error
	output   io.Writer
	bw       *bufio.Writer
	stopChan chan struct{}
	flushed  chan struct{}
	strategy OverflowStrategy
	refCount atomic.Int64
	lastErr  error
}

func newWorker(output io.Writer, cap int, strategy OverflowStrategy) *worker {
	w := &worker{
		queue:    make(chan *buffer, cap),
		syncChan: make(chan chan error),
		output:   output,
		bw:       bufio.NewWriterSize(output, 64*1024), // 64KB buffer
		stopChan: make(chan struct{}),
		flushed:  make(chan struct{}),
		strategy: strategy,
	}
	w.refCount.Store(1)
	w.start()

	_workersMu.Lock()
	_workers = append(_workers, w)
	_workersMu.Unlock()

	return w
}

func (w *worker) start() {
	go w.run()
}

func (w *worker) stop() {
	_workersMu.Lock()
	for i, worker := range _workers {
		if worker == w {
			_workers = append(_workers[:i], _workers[i+1:]...)
			break
		}
	}
	_workersMu.Unlock()

	close(w.stopChan)
	<-w.flushed
}

func (w *worker) submit(b *buffer) {
	select {
	case w.queue <- b:
		return
	default:
		// Fall through to overflow handling
	}

	switch w.strategy {
	case OverflowDrop:
		putBuffer(b)
	case OverflowBlock:
		w.queue <- b
	case OverflowSync:
		// Write directly to output
		w.output.Write(b.B)
		putBuffer(b)
	}
}

// sync pauses the calling goroutine until the worker writes all queued logs to the underlying writer.
func (w *worker) sync() error {
	errChan := make(chan error, 1)
	select {
	case w.syncChan <- errChan:
		return <-errChan
	case <-w.flushed:
		return nil
	}
}

// flush pauses the calling goroutine until the queue drains.
//
// Deprecated: Use sync instead. This method does not guarantee that the logs
// are actually written to the underlying writer.
func (w *worker) flush() {
	w.sync()
}

func (w *worker) run() {
	defer close(w.flushed)

	for {
		select {
		case <-w.stopChan:
			w.drainAll()
			w.flushBuffer()
			return
		case errChan := <-w.syncChan:
			w.drainAll()
			err := w.flushBuffer()
			errChan <- err
		case b := <-w.queue:
			w.write(b)

			// Batching: try to drain more from the channel without blocking
			for {
				select {
				case next := <-w.queue:
					w.write(next)
				default:
					goto flush
				}
			}
		flush:
			w.flushBuffer()
		}
	}
}

func (w *worker) drainAll() {
	for {
		select {
		case b := <-w.queue:
			w.write(b)
		default:
			return
		}
	}
}

func (w *worker) write(b *buffer) {
	if _, err := w.bw.Write(b.B); err != nil {
		w.handleError(err)
	}
	putBuffer(b)
}

func (w *worker) flushBuffer() error {
	if err := w.bw.Flush(); err != nil {
		w.handleError(err)
		return err
	}
	return nil
}

func (w *worker) handleError(err error) {
	if err != nil && w.lastErr != err {
		// Prevent log spam about logging errors
		w.lastErr = err
		fmt.Fprintf(os.Stderr, "velo: logging error: %v\n", err)
	}
}
