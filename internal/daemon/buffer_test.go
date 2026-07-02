package daemon

import (
	"sync"
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	buf := NewRingBuffer[int](5, nil)

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
	buf := NewRingBuffer[int](3, nil)

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
	buf := NewRingBuffer[int](5, nil)

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
	buf := NewRingBuffer[int](5, nil)

	items := buf.All()
	if items != nil {
		t.Errorf("expected nil for empty buffer, got %v", items)
	}
}

func TestRingBuffer_SingleElement(t *testing.T) {
	buf := NewRingBuffer[int](1, nil)

	buf.Push(1)
	buf.Push(2) // Overwrites 1

	items := buf.All()
	expected := []int{2}
	if !slicesEqual(items, expected) {
		t.Errorf("expected %v, got %v", expected, items)
	}
}

func TestRingBuffer_ZeroCapacity(t *testing.T) {
	buf := NewRingBuffer[int](0, nil)

	// Should default to capacity of 1
	if buf.Cap() != 1 {
		t.Errorf("expected cap 1 for zero input, got %d", buf.Cap())
	}
}

func TestRingBuffer_Concurrent(t *testing.T) {
	buf := NewRingBuffer[int](100, nil)

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
	buf := NewRingBuffer[int](5, nil)

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
	buf := NewRingBuffer[int](10, nil)

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
	buf := NewRingBuffer[int](10, nil)

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
	buf := NewRingBuffer[int](10, nil)

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
	buf := NewRingBuffer[int](10, nil)

	// Should not panic on empty buffer
	buf.RemoveIf(func(v *int) bool {
		return true
	})

	if buf.Len() != 0 {
		t.Errorf("expected len 0, got %d", buf.Len())
	}
}

// seqItem is a stamped element type for exercising the buffer's sequence
// assignment. val distinguishes entries; seq receives the stamped identifier.
type seqItem struct {
	val int
	seq uint64
}

func stampSeqItem(e *seqItem, s uint64) { e.seq = s }

func newSeqBuffer(capacity int) *RingBuffer[seqItem] {
	return NewRingBuffer(capacity, stampSeqItem)
}

func TestRingBuffer_SeqMonotonicFromOne(t *testing.T) {
	buf := newSeqBuffer(5)

	buf.Push(seqItem{val: 10})
	buf.Push(seqItem{val: 20})
	buf.Push(seqItem{val: 30})

	items := buf.All()
	want := []uint64{1, 2, 3}
	for i, it := range items {
		if it.seq != want[i] {
			t.Errorf("item %d: expected seq %d, got %d", i, want[i], it.seq)
		}
	}
}

func TestRingBuffer_SeqStableAcrossAll(t *testing.T) {
	buf := newSeqBuffer(5)

	buf.Push(seqItem{val: 10})
	buf.Push(seqItem{val: 20})

	first := buf.All()
	second := buf.All()
	if len(first) != len(second) {
		t.Fatalf("length mismatch: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].seq != second[i].seq {
			t.Errorf("item %d: seq not stable across All(): %d vs %d", i, first[i].seq, second[i].seq)
		}
	}
}

func TestRingBuffer_SeqSurvivesOverflow(t *testing.T) {
	buf := newSeqBuffer(3)

	// Six pushes into a capacity-3 buffer evicts the first three.
	for i := 1; i <= 6; i++ {
		buf.Push(seqItem{val: i})
	}

	items := buf.All()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	lowest, highest := items[0].seq, items[len(items)-1].seq
	if lowest != 4 {
		t.Errorf("expected lowest seq 4 after wrap, got %d", lowest)
	}
	if highest != 6 {
		t.Errorf("expected highest seq 6 (total pushes), got %d", highest)
	}
	// The gap between highest and count reveals the evicted count.
	if evicted := highest - uint64(len(items)); evicted != 3 {
		t.Errorf("expected 3 evicted entries visible from range, got %d", evicted)
	}
}

func TestRingBuffer_SeqSurvivesRemoveIf(t *testing.T) {
	buf := newSeqBuffer(10)

	for i := 1; i <= 5; i++ {
		buf.Push(seqItem{val: i})
	}

	// Remove even values; survivors are the entries pushed 1st, 3rd, 5th.
	buf.RemoveIf(func(it *seqItem) bool {
		return it.val%2 == 0
	})

	items := buf.All()
	wantVal := []int{1, 3, 5}
	wantSeq := []uint64{1, 3, 5}
	if len(items) != len(wantVal) {
		t.Fatalf("expected %d survivors, got %d", len(wantVal), len(items))
	}
	for i, it := range items {
		if it.val != wantVal[i] || it.seq != wantSeq[i] {
			t.Errorf("survivor %d: expected val=%d seq=%d, got val=%d seq=%d",
				i, wantVal[i], wantSeq[i], it.val, it.seq)
		}
	}
}

func TestRingBuffer_SeqResetsAfterClear(t *testing.T) {
	buf := newSeqBuffer(5)

	buf.Push(seqItem{val: 10})
	buf.Push(seqItem{val: 20})
	buf.Clear()

	buf.Push(seqItem{val: 30})
	items := buf.All()
	if len(items) != 1 {
		t.Fatalf("expected 1 item after clear+push, got %d", len(items))
	}
	if items[0].seq != 1 {
		t.Errorf("expected seq 1 after clear, got %d", items[0].seq)
	}
}

func TestRingBuffer_SeqConcurrentNoGapsOrCollisions(t *testing.T) {
	const writers, perWriter = 8, 250
	total := writers * perWriter

	// Sized to retain every push so All() exposes the full assigned set.
	buf := newSeqBuffer(total)

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				buf.Push(seqItem{val: base*perWriter + i})
			}
		}(w)
	}
	wg.Wait()

	items := buf.All()
	if len(items) != total {
		t.Fatalf("expected %d items, got %d", total, len(items))
	}

	// Concurrent pushes must assign exactly {1..total}: no value skipped, none
	// handed out twice.
	seen := make(map[uint64]bool, total)
	for _, it := range items {
		if it.seq == 0 {
			t.Errorf("entry val=%d was never stamped", it.val)
		}
		if seen[it.seq] {
			t.Errorf("seq %d assigned more than once", it.seq)
		}
		seen[it.seq] = true
	}
	for s := uint64(1); s <= uint64(total); s++ {
		if !seen[s] {
			t.Errorf("seq %d was skipped", s)
		}
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
