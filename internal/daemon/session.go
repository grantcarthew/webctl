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
}

// SessionManager tracks CDP page sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*session // keyed by sessionID
	activeID string              // currently active session ID
	order    []string            // session IDs in attachment order (newest last)
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*session),
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

	return &ipc.PageSession{
		ID:     s.SessionID,
		Title:  s.Title,
		URL:    s.URL,
		Active: true,
	}
}

// Get returns a session by ID, or nil if not found.
func (m *SessionManager) Get(sessionID string) *ipc.PageSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, exists := m.sessions[sessionID]
	if !exists {
		return nil
	}

	return &ipc.PageSession{
		ID:     s.SessionID,
		Title:  s.Title,
		URL:    s.URL,
		Active: s.SessionID == m.activeID,
	}
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
