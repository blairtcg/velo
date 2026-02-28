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
	"sync/atomic"
	"time"
)

const (
	_minLevel         = DebugLevel
	_maxLevel         = FatalLevel
	_numLevels        = _maxLevel - _minLevel + 1
	_countersPerLevel = 4096
)

type counter struct {
	resetAt atomic.Int64
	counter atomic.Uint64
}

type counters [_numLevels][_countersPerLevel]counter

func newCounters() *counters {
	return &counters{}
}

func (cs *counters) get(lvl Level, key string) *counter {
	i := lvl - _minLevel
	if i < 0 || i >= _numLevels {
		// Fallback for invalid levels
		i = 0
	}
	j := fnv32a(key) % _countersPerLevel
	return &cs[i][j]
}

// fnv32a, adapted from "hash/fnv", but without a []byte(string) alloc
func fnv32a(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= prime32
	}
	return hash
}

func (c *counter) IncCheckReset(t time.Time, tick time.Duration) uint64 {
	tn := t.UnixNano()
	resetAfter := c.resetAt.Load()
	if resetAfter > tn {
		return c.counter.Add(1)
	}

	c.counter.Store(1)

	newResetAfter := tn + tick.Nanoseconds()
	if !c.resetAt.CompareAndSwap(resetAfter, newResetAfter) {
		// We raced with another goroutine trying to reset, and it also reset
		// the counter to 1, so we need to reincrement the counter.
		return c.counter.Add(1)
	}

	return 1
}

// SamplingDecision represents a decision made by the sampler as a bit field.
//
// Future versions may add more decision types.
type SamplingDecision uint32

const (
	// LogDropped indicates that the Sampler discarded a log entry.
	LogDropped SamplingDecision = 1 << iota
	// LogSampled indicates that the Sampler allowed a log entry through.
	LogSampled
)

// optionFunc wraps a func so it satisfies the SamplerOption interface.
type optionFunc func(*sampler)

func (f optionFunc) apply(s *sampler) {
	f(s)
}

// SamplerOption configures the behavior of a Sampler.
type SamplerOption interface {
	apply(*sampler)
}

// nopSamplingHook is the default hook used by sampler.
func nopSamplingHook(Level, string, SamplingDecision) {}

// SamplerHook registers a callback function that fires whenever the Sampler makes a decision.
//
// Use this hook to monitor sampler performance. For example, you can track metrics
// comparing dropped versus sampled logs.
//
//	var dropped atomic.Int64
//	velo.SamplerHook(func(lvl velo.Level, msg string, dec velo.SamplingDecision) {
//	  if dec&velo.LogDropped > 0 {
//	    dropped.Inc()
//	  }
//	})
func SamplerHook(hook func(lvl Level, msg string, dec SamplingDecision)) SamplerOption {
	return optionFunc(func(s *sampler) {
		s.hook = hook
	})
}

// NewSamplerWithOptions creates a new Logger that samples incoming entries.
//
// Sampling caps the CPU and I/O load of logging while preserving a representative
// subset of your logs. It samples by allowing the first N entries with a specific
// level and message through during each tick. If the Logger sees more entries with
// the same level and message in that interval, it logs every Mth message and drops
// the rest.
//
// For example:
//
//	logger = NewSamplerWithOptions(logger, time.Second, 10, 5)
//
// This configuration logs the first 10 identical entries in a one second interval
// as is. After that, it allows every 5th identical entry through. If `thereafter`
// is zero, the Logger drops all subsequent identical entries in that interval.
//
// You can configure the Sampler to report its decisions using the SamplerHook option.
//
// Performance Note: The sampling implementation prioritizes speed over absolute
// precision. Under heavy load, each tick may slightly over sample or under sample.
func NewSamplerWithOptions(logger *Logger, tick time.Duration, first, thereafter int, opts ...SamplerOption) *Logger {
	s := &sampler{
		tick:       tick,
		counts:     newCounters(),
		first:      uint64(first),
		thereafter: uint64(thereafter),
		hook:       nopSamplingHook,
	}
	for _, opt := range opts {
		opt.apply(s)
	}

	nl := &Logger{
		fields:      logger.fields,
		typedFields: logger.typedFields,
		worker:      logger.worker,
		level:       logger.level,
		sampler:     s,
	}
	nl.config.Store(logger.config.Load())
	logger.worker.refCount.Add(1)
	return nl
}

// NewSampler creates a Logger that samples incoming entries.
//
// Sampling caps the CPU and I/O load of logging while preserving a representative
// subset of your logs. It samples by allowing the first N entries with a specific
// level and message through during each tick. If the Logger sees more entries with
// the same level and message in that interval, it logs every Mth message and drops
// the rest.
//
// Performance Note: The sampling implementation prioritizes speed over absolute
// precision. Under heavy load, each tick may slightly over sample or under sample.
//
// Deprecated: Use NewSamplerWithOptions instead.
func NewSampler(logger *Logger, tick time.Duration, first, thereafter int) *Logger {
	return NewSamplerWithOptions(logger, tick, first, thereafter)
}

type sampler struct {
	counts            *counters
	tick              time.Duration
	first, thereafter uint64
	hook              func(Level, string, SamplingDecision)
}

func (s *sampler) check(lvl Level, msg string, t time.Time) bool {
	if lvl >= _minLevel && lvl <= _maxLevel {
		counter := s.counts.get(lvl, msg)
		n := counter.IncCheckReset(t, s.tick)
		if n > s.first && (s.thereafter == 0 || (n-s.first)%s.thereafter != 0) {
			s.hook(lvl, msg, LogDropped)
			return false
		}
		s.hook(lvl, msg, LogSampled)
	}
	return true
}
