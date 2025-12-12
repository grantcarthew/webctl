package daemon

import "sync"

// RingBuffer is a thread-safe circular buffer with fixed capacity.
// When the buffer is full, new items overwrite the oldest items.
type RingBuffer[T any] struct {
	items []T
	head  int  // next write position
	count int  // number of items currently in buffer
	cap   int  // maximum capacity
	mu    sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 1
	}
	return &RingBuffer[T]{
		items: make([]T, capacity),
		cap:   capacity,
	}
}

// Push adds an item to the buffer.
// If the buffer is full, the oldest item is overwritten.
func (b *RingBuffer[T]) Push(item T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.items[b.head] = item
	b.head = (b.head + 1) % b.cap

	if b.count < b.cap {
		b.count++
	}
}

// All returns all items in the buffer, oldest first.
// Allocates a new slice on each call. This is acceptable for the current
// request-response IPC pattern where each query is a discrete operation.
func (b *RingBuffer[T]) All() []T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.count == 0 {
		return nil
	}

	result := make([]T, b.count)

	// Calculate the start position (oldest item)
	start := 0
	if b.count == b.cap {
		start = b.head // head points to oldest when full
	}

	for i := 0; i < b.count; i++ {
		idx := (start + i) % b.cap
		result[i] = b.items[idx]
	}

	return result
}

// Len returns the current number of items in the buffer.
func (b *RingBuffer[T]) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

// Cap returns the buffer capacity.
func (b *RingBuffer[T]) Cap() int {
	return b.cap
}

// Update iterates through buffer items from newest to oldest,
// calling fn with a pointer to each item. Iteration stops when fn returns true.
// This allows in-place modification of buffer entries.
func (b *RingBuffer[T]) Update(fn func(*T) bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return
	}

	// Iterate from newest to oldest
	for i := 0; i < b.count; i++ {
		idx := (b.head - 1 - i + b.cap) % b.cap
		if fn(&b.items[idx]) {
			return
		}
	}
}

// Clear removes all items from the buffer.
func (b *RingBuffer[T]) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Zero out items to allow GC
	var zero T
	for i := range b.items {
		b.items[i] = zero
	}

	b.head = 0
	b.count = 0
}
