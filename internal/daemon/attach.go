package daemon

import "sync"

// attachSet deduplicates Target.attachToTarget calls by targetID. mark reports
// whether the caller is the first to claim a targetID; a failed attach calls clear
// so a retry can occur. This is attach-process bookkeeping, kept separate from the
// navigation rendezvous and from session identity.
type attachSet struct {
	mu  sync.Mutex
	ids map[string]struct{}
}

// newAttachSet creates an empty attach set.
func newAttachSet() *attachSet {
	return &attachSet{ids: make(map[string]struct{})}
}

// mark records targetID as attaching, returning true only for the first claim.
func (s *attachSet) mark(targetID string) (first bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ids[targetID]; ok {
		return false
	}
	s.ids[targetID] = struct{}{}
	return true
}

// clear removes targetID so a failed attach can be retried.
func (s *attachSet) clear(targetID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ids, targetID)
}
