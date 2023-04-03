// Package doublebuf provides a generic double-buffering mechanism.
// Use this for read-optimized double buffering.
package doublebuf

import (
	"context"
	"sync/atomic"
)

// DoubleBuffer is a double buffering implementation.
type DoubleBuffer[T comparable] struct {
	a, b        T
	back, front *T
	next        atomic.Value
	prev        chan *T
}

func New[T comparable](a, b T) *DoubleBuffer[T] {
	db := &DoubleBuffer[T]{
		a: a, b: b,
		prev: make(chan *T, 1),
	}
	db.back = &db.a
	db.front = &db.b
	db.next.Store((*T)(nil))
	return db
}

// Back returns the next back buffer.
// Back will return the same value until Ready is called.
// Back is safe to call concurrently with Next and Front.
// Back is not safe to call concurrently with Ready.
// Calling Back multiple times is idempotent.
func (db *DoubleBuffer[T]) Back(ctx context.Context) (*T, error) {
	if db.back == nil { // db.back has been submitted via a previous call to ready
		// wait for the consumer to replace the back buffer
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case db.back = <-db.prev:
		}
	}
	return db.back, nil
}

// Ready is used to signal that the back buffer is ready to be swapped with
// the front buffer in the next call to Next.
// It is safe to call Ready concurrently with Next and Front.
// It is not safe to call Ready concurrently with Back.
// Calling Ready multiple times is idempotent.
func (db *DoubleBuffer[T]) Ready() {
	if db.back != nil {
		db.next.Store(db.back)
		db.back = nil
	}
}

// Front returns the front buffer.
func (db *DoubleBuffer[T]) Front() T { return *db.front }

// Next swaps the front and back buffers and returns the new front buffer
// if the back buffer is ready to be used. Otherwise, it returns the
// current front buffer. The boolean return value changed is true if the
// front buffer was swapped, and false otherwise.
// It is safe to call Next concurrently, however, an old reference to the
// front buffer is no longer guaranteed to be valid if Next returns with changed set to true.
func (db *DoubleBuffer[T]) Next() (t T, changed bool) {
	// The sequence:
	// 1. Check if a new buffer is ready.
	// 2. If not, return the current front buffer.
	// 3. If so, make the new buffer available for swapping.
	next := db.next.Swap((*T)(nil)).(*T)
	if next != nil {
		db.prev <- db.front
		db.front = next
	}
	return *db.front, next != nil
}
