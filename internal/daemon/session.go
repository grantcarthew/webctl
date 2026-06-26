package daemon

import (
	"strings"
	"sync"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// session holds internal session state (extends ipc.PageSession with targetID).
type session struct {
	SessionID string
	TargetID  string
	URL       string
	Title     string
	// networkEnabled records that Network.enable succeeded for this session.
	// It is a fact about the session, so it lives here rather than in a map on
	// the daemon, and it gates the at-most-once Network.enable guarantee.
	networkEnabled bool
}

// SessionManager tracks CDP page sessions and the tab attach/detach rendezvous.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*session // keyed by sessionID
	activeID string              // currently active session ID
	order    []string            // session IDs in attachment order (newest last)

	// Tab rendezvous waiters, signalled by Add/Remove under mu. Registering a
	// waiter and the fast-path state check are one locked operation (see
	// waitForAttach/waitForDetach), so Add/Remove cannot signal before the
	// waiter exists.
	attachWaiters map[string]chan struct{} // targetID -> closed when its session attaches
	detachWaiters map[string]chan struct{} // sessionID -> closed when the session detaches
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:      make(map[string]*session),
		attachWaiters: make(map[string]chan struct{}),
		detachWaiters: make(map[string]chan struct{}),
	}
}

// Add adds a new session. If it's the first session, it becomes active.
func (m *SessionManager) Add(sessionID, targetID, url, title string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[sessionID] = &session{
		SessionID: sessionID,
		TargetID:  targetID,
		URL:       url,
		Title:     title,
	}
	m.order = append(m.order, sessionID)

	// First session becomes active
	if m.activeID == "" {
		m.activeID = sessionID
	}

	// Signal any tab-new waiter for this targetID.
	if ch, ok := m.attachWaiters[targetID]; ok {
		close(ch)
		delete(m.attachWaiters, targetID)
	}
}

// Remove removes a session. If it was active, switches to most recent remaining.
// Returns the new active session ID (empty if none remain) and true if active changed.
func (m *SessionManager) Remove(sessionID string) (newActiveID string, activeChanged bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return m.activeID, false
	}

	delete(m.sessions, sessionID)

	// Remove from order
	for i, id := range m.order {
		if id == sessionID {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}

	// Signal any tab-close waiter for this sessionID.
	if ch, ok := m.detachWaiters[sessionID]; ok {
		close(ch)
		delete(m.detachWaiters, sessionID)
	}

	// If active session was removed, switch to most recent
	if m.activeID == sessionID {
		if len(m.order) > 0 {
			m.activeID = m.order[len(m.order)-1]
		} else {
			m.activeID = ""
		}
		return m.activeID, true
	}

	return m.activeID, false
}

// Update updates a session's URL and title by session ID.
func (m *SessionManager) Update(sessionID, url, title string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, exists := m.sessions[sessionID]; exists {
		if url != "" {
			s.URL = url
		}
		if title != "" {
			s.Title = title
		}
	}
}

// UpdateByTargetID updates a session's URL and title by target ID.
func (m *SessionManager) UpdateByTargetID(targetID, url, title string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.sessions {
		if s.TargetID == targetID {
			if url != "" {
				s.URL = url
			}
			if title != "" {
				s.Title = title
			}
			return
		}
	}
}

// SetActive sets the active session by ID.
// Returns false if the session doesn't exist.
func (m *SessionManager) SetActive(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return false
	}
	m.activeID = sessionID
	return true
}

// ActiveID returns the current active session ID (empty if none).
func (m *SessionManager) ActiveID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeID
}

// Active returns the active session info, or nil if none.
func (m *SessionManager) Active() *ipc.PageSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeID == "" {
		return nil
	}

	s, exists := m.sessions[m.activeID]
	if !exists {
		return nil
	}

	return m.toPageSessionLocked(s)
}

// Get returns a session by ID, or nil if not found.
func (m *SessionManager) Get(sessionID string) *ipc.PageSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, exists := m.sessions[sessionID]
	if !exists {
		return nil
	}

	return m.toPageSessionLocked(s)
}

// TargetID returns the targetID for the given sessionID, or empty string if not found.
func (m *SessionManager) TargetID(sessionID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if s, exists := m.sessions[sessionID]; exists {
		return s.TargetID
	}
	return ""
}

// GetByTargetID returns the session matching the given targetID, or nil if not found.
func (m *SessionManager) GetByTargetID(targetID string) *ipc.PageSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		if s.TargetID == targetID {
			return m.toPageSessionLocked(s)
		}
	}
	return nil
}

// All returns all sessions as IPC PageSession list.
func (m *SessionManager) All() []ipc.PageSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ipc.PageSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, ipc.PageSession{
			ID:     s.SessionID,
			Title:  s.Title,
			URL:    s.URL,
			Active: s.SessionID == m.activeID,
		})
	}
	return result
}

// Count returns the number of sessions.
func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Clear removes all sessions and resets the manager state.
// Used when browser connection is lost.
func (m *SessionManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*session)
	m.activeID = ""
	m.order = nil
}

// ClaimNetworkEnable marks the session's Network domain as enabled if it was not
// already, returning true only for the caller that wins the claim. The winner
// performs Network.enable outside the lock and, on failure, calls
// ClearNetworkEnabled so a later caller can retry. Returns false if the session
// is unknown.
func (m *SessionManager) ClaimNetworkEnable(sessionID string) (first bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionID]
	if !ok || s.networkEnabled {
		return false
	}
	s.networkEnabled = true
	return true
}

// ClearNetworkEnabled clears the session's Network-enabled flag so a failed
// enable can be retried.
func (m *SessionManager) ClearNetworkEnabled(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[sessionID]; ok {
		s.networkEnabled = false
	}
}

// NetworkEnabled reports whether Network.enable succeeded for the session.
func (m *SessionManager) NetworkEnabled(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if s, ok := m.sessions[sessionID]; ok {
		return s.networkEnabled
	}
	return false
}

// waitForAttach atomically resolves the tab-new rendezvous for targetID. If the
// session already attached it returns the session and a nil channel (fast path);
// otherwise it registers a waiter and returns a nil session with the channel,
// which Add closes when the session attaches. Exactly one of the return values is
// non-nil. The caller must stopWaitForAttach when done to release a registered
// waiter that never fired.
func (m *SessionManager) waitForAttach(targetID string) (*ipc.PageSession, <-chan struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.sessions {
		if s.TargetID == targetID {
			return m.toPageSessionLocked(s), nil
		}
	}

	ch := make(chan struct{})
	m.attachWaiters[targetID] = ch
	return nil, ch
}

// stopWaitForAttach removes a targetID attach waiter if it is still registered.
func (m *SessionManager) stopWaitForAttach(targetID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.attachWaiters, targetID)
}

// waitForDetach atomically resolves the tab-close rendezvous for sessionID. If
// the session is still present it registers a waiter and returns the channel,
// which Remove closes when the session detaches; if the session is already gone
// it returns nil (fast path, detach already fired). The caller must
// stopWaitForDetach when done to release a registered waiter that never fired.
func (m *SessionManager) waitForDetach(sessionID string) <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[sessionID]; !ok {
		return nil
	}

	ch := make(chan struct{})
	m.detachWaiters[sessionID] = ch
	return ch
}

// stopWaitForDetach removes a sessionID detach waiter if it is still registered.
func (m *SessionManager) stopWaitForDetach(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.detachWaiters, sessionID)
}

// toPageSessionLocked builds the IPC view of a session. Callers must hold m.mu.
func (m *SessionManager) toPageSessionLocked(s *session) *ipc.PageSession {
	return &ipc.PageSession{
		ID:     s.SessionID,
		Title:  s.Title,
		URL:    s.URL,
		Active: s.SessionID == m.activeID,
	}
}

// FindByQuery searches for sessions matching the query.
// Query is matched against session ID prefix (case-sensitive) or title substring (case-insensitive).
// Returns matching sessions.
func (m *SessionManager) FindByQuery(query string) []ipc.PageSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if query == "" {
		return nil
	}

	var matches []ipc.PageSession

	// First try exact session ID prefix match
	for _, s := range m.sessions {
		if len(s.SessionID) >= len(query) && s.SessionID[:len(query)] == query {
			matches = append(matches, ipc.PageSession{
				ID:     s.SessionID,
				Title:  s.Title,
				URL:    s.URL,
				Active: s.SessionID == m.activeID,
			})
		}
	}

	if len(matches) > 0 {
		return matches
	}

	// Fall back to case-insensitive title substring match
	queryLower := strings.ToLower(query)
	for _, s := range m.sessions {
		if strings.Contains(strings.ToLower(s.Title), queryLower) {
			matches = append(matches, ipc.PageSession{
				ID:     s.SessionID,
				Title:  s.Title,
				URL:    s.URL,
				Active: s.SessionID == m.activeID,
			})
		}
	}

	return matches
}
