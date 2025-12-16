package daemon

import (
	"testing"
)

func TestSessionManager_Add(t *testing.T) {
	sm := NewSessionManager()

	// First session becomes active
	sm.Add("sess1", "target1", "http://example.com", "Example")
	if sm.ActiveID() != "sess1" {
		t.Errorf("expected active session 'sess1', got '%s'", sm.ActiveID())
	}
	if sm.Count() != 1 {
		t.Errorf("expected 1 session, got %d", sm.Count())
	}

	// Second session doesn't change active
	sm.Add("sess2", "target2", "http://other.com", "Other")
	if sm.ActiveID() != "sess1" {
		t.Errorf("expected active session still 'sess1', got '%s'", sm.ActiveID())
	}
	if sm.Count() != 2 {
		t.Errorf("expected 2 sessions, got %d", sm.Count())
	}
}

func TestSessionManager_Remove(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("sess1", "target1", "http://example.com", "Example")
	sm.Add("sess2", "target2", "http://other.com", "Other")

	// Remove non-active session
	newActive, changed := sm.Remove("sess2")
	if changed {
		t.Error("expected no active change when removing non-active session")
	}
	if newActive != "sess1" {
		t.Errorf("expected active still 'sess1', got '%s'", newActive)
	}
	if sm.Count() != 1 {
		t.Errorf("expected 1 session, got %d", sm.Count())
	}

	// Remove active session - should switch to remaining
	sm.Add("sess3", "target3", "http://third.com", "Third")
	newActive, changed = sm.Remove("sess1")
	if !changed {
		t.Error("expected active change when removing active session")
	}
	if newActive != "sess3" {
		t.Errorf("expected active switched to 'sess3', got '%s'", newActive)
	}
}

func TestSessionManager_SetActive(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("sess1", "target1", "http://example.com", "Example")
	sm.Add("sess2", "target2", "http://other.com", "Other")

	// Switch to existing session
	if !sm.SetActive("sess2") {
		t.Error("expected SetActive to return true for existing session")
	}
	if sm.ActiveID() != "sess2" {
		t.Errorf("expected active 'sess2', got '%s'", sm.ActiveID())
	}

	// Try to switch to non-existent session
	if sm.SetActive("nonexistent") {
		t.Error("expected SetActive to return false for non-existent session")
	}
	if sm.ActiveID() != "sess2" {
		t.Errorf("expected active still 'sess2', got '%s'", sm.ActiveID())
	}
}

func TestSessionManager_Update(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("sess1", "target1", "http://example.com", "Example")

	sm.Update("sess1", "http://updated.com", "Updated Title")

	session := sm.Get("sess1")
	if session.URL != "http://updated.com" {
		t.Errorf("expected URL 'http://updated.com', got '%s'", session.URL)
	}
	if session.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", session.Title)
	}
}

func TestSessionManager_UpdateByTargetID(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("sess1", "target1", "http://example.com", "Example")

	sm.UpdateByTargetID("target1", "http://updated.com", "Updated Title")

	session := sm.Get("sess1")
	if session.URL != "http://updated.com" {
		t.Errorf("expected URL 'http://updated.com', got '%s'", session.URL)
	}
	if session.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", session.Title)
	}
}

func TestSessionManager_FindByQuery(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("ABCD1234", "target1", "http://example.com", "Example Domain")
	sm.Add("EFGH5678", "target2", "http://other.com", "Other Page")

	// Match by session ID prefix
	matches := sm.FindByQuery("ABCD")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match by ID prefix, got %d", len(matches))
	}
	if matches[0].ID != "ABCD1234" {
		t.Errorf("expected ID 'ABCD1234', got '%s'", matches[0].ID)
	}

	// Match by title substring (case-insensitive)
	matches = sm.FindByQuery("other")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match by title, got %d", len(matches))
	}
	if matches[0].ID != "EFGH5678" {
		t.Errorf("expected ID 'EFGH5678', got '%s'", matches[0].ID)
	}

	// No match
	matches = sm.FindByQuery("nonexistent")
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %d", len(matches))
	}
}

func TestSessionManager_Active(t *testing.T) {
	sm := NewSessionManager()

	// No active session initially
	if sm.Active() != nil {
		t.Error("expected nil active session initially")
	}

	sm.Add("sess1", "target1", "http://example.com", "Example")

	active := sm.Active()
	if active == nil {
		t.Fatal("expected active session after add")
	}
	if active.ID != "sess1" {
		t.Errorf("expected active ID 'sess1', got '%s'", active.ID)
	}
	if !active.Active {
		t.Error("expected Active field to be true")
	}
}

func TestSessionManager_All(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("sess1", "target1", "http://example.com", "Example")
	sm.Add("sess2", "target2", "http://other.com", "Other")

	all := sm.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}

	// Check that one is marked active
	activeCount := 0
	for _, s := range all {
		if s.Active {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("expected 1 active session, got %d", activeCount)
	}
}
