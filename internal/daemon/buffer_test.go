package daemon

import (
	"sync"
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	buf := NewRingBuffer[int](5)

	if buf.Len() != 0 {
		t.Errorf("expected len 0, got %d", buf.Len())
	}
	if buf.Cap() != 5 {
		t.Errorf("expected cap 5, got %d", buf.Cap())
	}

	buf.Push(1)
	buf.Push(2)
	buf.Push(3)

	if buf.Len() != 3 {
		t.Errorf("expected len 3, got %d", buf.Len())
	}

	items := buf.All()
	expected := []int{1, 2, 3}
	if !slicesEqual(items, expected) {
		t.Errorf("expected %v, got %v", expected, items)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	buf := NewRingBuffer[int](3)

	buf.Push(1)
	buf.Push(2)
	buf.Push(3)
	buf.Push(4) // Overwrites 1
	buf.Push(5) // Overwrites 2

	if buf.Len() != 3 {
		t.Errorf("expected len 3, got %d", buf.Len())
	}

	items := buf.All()
	expected := []int{3, 4, 5}
	if !slicesEqual(items, expected) {
		t.Errorf("expected %v, got %v", expected, items)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	buf := NewRingBuffer[int](5)

	buf.Push(1)
	buf.Push(2)
	buf.Push(3)
	buf.Clear()

	if buf.Len() != 0 {
		t.Errorf("expected len 0 after clear, got %d", buf.Len())
	}

	items := buf.All()
	if items != nil {
		t.Errorf("expected nil after clear, got %v", items)
	}

	// Verify we can push after clear
	buf.Push(10)
	if buf.Len() != 1 {
		t.Errorf("expected len 1 after push, got %d", buf.Len())
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	buf := NewRingBuffer[int](5)

	items := buf.All()
	if items != nil {
		t.Errorf("expected nil for empty buffer, got %v", items)
	}
}

func TestRingBuffer_SingleElement(t *testing.T) {
	buf := NewRingBuffer[int](1)

	buf.Push(1)
	buf.Push(2) // Overwrites 1

	items := buf.All()
	expected := []int{2}
	if !slicesEqual(items, expected) {
		t.Errorf("expected %v, got %v", expected, items)
	}
}

func TestRingBuffer_ZeroCapacity(t *testing.T) {
	buf := NewRingBuffer[int](0)

	// Should default to capacity of 1
	if buf.Cap() != 1 {
		t.Errorf("expected cap 1 for zero input, got %d", buf.Cap())
	}
}

func TestRingBuffer_Concurrent(t *testing.T) {
	buf := NewRingBuffer[int](100)

	var wg sync.WaitGroup
	n := 10

	// Writers
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf.Push(base*100 + j)
			}
		}(i)
	}

	// Readers
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = buf.All()
				_ = buf.Len()
			}
		}()
	}

	wg.Wait()

	// Just verify no deadlocks or panics occurred
	if buf.Len() > 100 {
		t.Errorf("buffer len exceeded capacity")
	}
}

func TestRingBuffer_ExactCapacity(t *testing.T) {
	buf := NewRingBuffer[int](5)

	// Fill exactly to capacity
	for i := 1; i <= 5; i++ {
		buf.Push(i)
	}

	items := buf.All()
	expected := []int{1, 2, 3, 4, 5}
	if !slicesEqual(items, expected) {
		t.Errorf("expected %v, got %v", expected, items)
	}

	// Add one more
	buf.Push(6)
	items = buf.All()
	expected = []int{2, 3, 4, 5, 6}
	if !slicesEqual(items, expected) {
		t.Errorf("expected %v, got %v", expected, items)
	}
}

func TestRingBuffer_RemoveIf(t *testing.T) {
	buf := NewRingBuffer[int](10)

	for i := 1; i <= 5; i++ {
		buf.Push(i)
	}

	// Remove even numbers
	buf.RemoveIf(func(v *int) bool {
		return *v%2 == 0
	})

	items := buf.All()
	expected := []int{1, 3, 5}
	if !slicesEqual(items, expected) {
		t.Errorf("expected %v, got %v", expected, items)
	}
}

func TestRingBuffer_RemoveIfAll(t *testing.T) {
	buf := NewRingBuffer[int](10)

	for i := 1; i <= 5; i++ {
		buf.Push(i)
	}

	// Remove all
	buf.RemoveIf(func(v *int) bool {
		return true
	})

	if buf.Len() != 0 {
		t.Errorf("expected len 0, got %d", buf.Len())
	}
}

func TestRingBuffer_RemoveIfNone(t *testing.T) {
	buf := NewRingBuffer[int](10)

	for i := 1; i <= 5; i++ {
		buf.Push(i)
	}

	// Remove none
	buf.RemoveIf(func(v *int) bool {
		return false
	})

	if buf.Len() != 5 {
		t.Errorf("expected len 5, got %d", buf.Len())
	}
}

func TestRingBuffer_RemoveIfEmpty(t *testing.T) {
	buf := NewRingBuffer[int](10)

	// Should not panic on empty buffer
	buf.RemoveIf(func(v *int) bool {
		return true
	})

	if buf.Len() != 0 {
		t.Errorf("expected len 0, got %d", buf.Len())
	}
}

func slicesEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
