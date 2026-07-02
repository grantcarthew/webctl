package daemon

import "sync"

// RingBuffer is a thread-safe circular buffer with fixed capacity.
// When the buffer is full, new items overwrite the oldest items.
type RingBuffer[T any] struct {
	items []T
	head  int // next write position
	count int // number of items currently in buffer
	cap   int // maximum capacity
	// seq is the last assigned sequence number. It increments before each
	// stamp, so the first push after construction or Clear assigns 1 and 0
	// remains reserved for entries that never passed through Push.
	seq uint64
	// stamp records an entry's assigned sequence number as it is pushed, or is
	// nil for element types without sequence identity. A function rather than a
	// setter constraint so the buffer stays generic over T (RingBuffer[int]).
	stamp func(*T, uint64)
	mu    sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity. stamp,
// if non-nil, is invoked under the write lock in Push to record each entry's
// assigned sequence number; pass nil for element types without a seq field.
func NewRingBuffer[T any](capacity int, stamp func(*T, uint64)) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 1
	}
	return &RingBuffer[T]{
		items: make([]T, capacity),
		cap:   capacity,
		stamp: stamp,
	}
}

// Push adds an item to the buffer.
// If the buffer is full, the oldest item is overwritten.
func (b *RingBuffer[T]) Push(item T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.items[b.head] = item
	if b.stamp != nil {
		b.seq++
		b.stamp(&b.items[b.head], b.seq)
	}
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
	b.seq = 0
}

// RemoveIf removes all items for which fn returns true.
// Items are compacted in-place, maintaining order.
func (b *RingBuffer[T]) RemoveIf(fn func(*T) bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return
	}

	// Collect items to keep
	var keep []T
	start := 0
	if b.count == b.cap {
		start = b.head
	}

	for i := 0; i < b.count; i++ {
		idx := (start + i) % b.cap
		if !fn(&b.items[idx]) {
			keep = append(keep, b.items[idx])
		}
	}

	// Zero out buffer
	var zero T
	for i := range b.items {
		b.items[i] = zero
	}

	// Re-add kept items
	b.head = 0
	b.count = len(keep)
	copy(b.items, keep)
	b.head = b.count % b.cap
}
