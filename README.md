# Velo

Fast, asynchronous, structured logging in Go.

Velo gives you a clear, developer friendly API without sacrificing app performance. It moves I/O operations to a background worker so your hot paths never block waiting for logs to write. 

## Installation

```bash
go get github.com/blairtcg/velo
```

## Quick start

Use the standard interface when you want clean, minimal code. It accepts loosely typed key-value pairs.

```go
logger := velo.New()
defer logger.Close() // flushes the buffer

logger.Info("failed to fetch URL",
  "url", url,
  "attempt", 3,
  "backoff", time.Second,
)
```

Use the `LogFields` API when performance and type safety are critical. This approach uses strongly typed fields to eliminate memory allocations caused by interface boxing.

```go
logger := velo.New()
defer logger.Close()

logger.LogFields(velo.InfoLevel, "failed to fetch URL",
  velo.String("url", url),
  velo.Int("attempt", 3),
  velo.Duration("backoff", time.Second),
)
```

If you use Go's standard structured logging library, you can configure Velo as your `slog.Handler`.

```go
logger := slog.New(velo.NewSlogHandler())
slog.SetDefault(logger)
```

## Performance and backpressure

Velo uses a hybrid synchronous and asynchronous model to keep your application fast under heavy load.

When you log a message, Velo formats it on the caller's goroutine and sends it to a buffered channel. A background worker reads from this channel and writes to the output stream. This parallelizes the work and isolates your application code from I/O latency.

In high throughput scenarios, your application might generate logs faster than the output stream can write them. When the buffer is full, backpressure occurs. 

You can control how Velo handles a full buffer using the `OverflowStrategy` option:

*   **`OverflowSync` (Default):** The logger temporarily switches to a synchronous write. The calling goroutine writes its preformatted log entry directly to the output stream. This prevents log loss and controls memory use, but temporarily blocks the calling goroutine.
*   **`OverflowDrop`:** The logger discards new log entries until space opens up in the buffer. Use this strategy when maintaining low latency is more critical than keeping every log entry.
*   **`OverflowBlock`:** The calling goroutine waits and blocks until space becomes available in the buffer.

## Output formats

You can configure the background worker to write logs in multiple output formats. Velo currently supports both normal easy to read text formatting for local development and JSON formatting for production logs.
