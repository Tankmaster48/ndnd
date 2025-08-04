package io

import (
	"bufio"
	"io"
	"sync"
	"time"
)

// TimedWriter is a buffered writer that flushes automatically
// when a deadline is set and the deadline is exceeded.
type TimedWriter struct {
	*bufio.Writer
	mutex    sync.Mutex
	deadline time.Duration
	maxQueue int

	queueSize int
	timer     *time.Timer
	prevErr   error
}

// Constructs a `TimedWriter` with the provided `io.Writer` and buffer size, initialized with a default write deadline of 1 millisecond and a maximum queue size of 8.
func NewTimedWriter(io io.Writer, bufsize int) *TimedWriter {
	return &TimedWriter{
		Writer:   bufio.NewWriterSize(io, bufsize),
		deadline: 1 * time.Millisecond,
		maxQueue: 8,
	}
}

// Sets the deadline for the TimedWriter to the specified duration, determining the maximum time allowed for write operations.
func (w *TimedWriter) SetDeadline(d time.Duration) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.deadline = d
}

// Sets the maximum queue size for the TimedWriter to the specified integer value, ensuring thread-safe updates with a mutex lock.
func (w *TimedWriter) SetMaxQueue(s int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.maxQueue = s
}

// Safely flushes any buffered data from the TimedWriter by acquiring a mutex lock before invoking the internal flush operation, returning any resulting errors.
func (w *TimedWriter) Flush() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.flush_()
}

// Writes data to an underlying writer, buffering it and flushing either when the buffer reaches the maximum queue size, the deadline elapses, or an error occurs, ensuring thread-safe and timed/batch-optimized output.
func (w *TimedWriter) Write(p []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if err := w.prevErr; err != nil {
		w.prevErr = nil
		return 0, err
	}

	n, err = w.Writer.Write(p)
	if err != nil {
		return n, err
	}

	w.queueSize++
	if w.deadline == 0 || w.queueSize >= w.maxQueue {
		return n, w.flush_()
	}

	if w.timer == nil {
		w.timer = time.AfterFunc(w.deadline, func() { w.Flush() })
	}

	return
}

// Flushes the underlying writer, stops any pending timer, and resets the queue size, returning any error encountered during the flush operation.
func (w *TimedWriter) flush_() error {
	err := w.Writer.Flush()
	if err != nil {
		w.prevErr = err
	}

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.queueSize = 0

	return err
}
