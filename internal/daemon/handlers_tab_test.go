package daemon

import (
	"encoding/json"
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
)

func TestSessionManager_TargetID(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("sess1", "target1", "http://example.com", "Example")
	sm.Add("sess2", "target2", "http://other.com", "Other")

	if got := sm.TargetID("sess1"); got != "target1" {
		t.Errorf("TargetID(sess1) = %q, want %q", got, "target1")
	}
	if got := sm.TargetID("sess2"); got != "target2" {
		t.Errorf("TargetID(sess2) = %q, want %q", got, "target2")
	}
	if got := sm.TargetID("missing"); got != "" {
		t.Errorf("TargetID(missing) = %q, want empty", got)
	}
}

func TestSessionManager_GetByTargetID(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("sess1", "target1", "http://example.com", "Example")
	sm.Add("sess2", "target2", "http://other.com", "Other")

	got := sm.GetByTargetID("target2")
	if got == nil {
		t.Fatal("GetByTargetID(target2) returned nil, expected session")
	}
	if got.ID != "sess2" {
		t.Errorf("session.ID = %q, want sess2", got.ID)
	}
	if got.URL != "http://other.com" {
		t.Errorf("session.URL = %q, want http://other.com", got.URL)
	}

	if sm.GetByTargetID("missing") != nil {
		t.Error("GetByTargetID(missing) should return nil")
	}
}

func TestAmbiguousTabError(t *testing.T) {
	matches := []ipc.PageSession{
		{ID: "abc12345", Title: "Test 1"},
		{ID: "def67890", Title: "Test 2"},
	}

	resp := ambiguousTabError("test", matches)

	if resp.OK {
		t.Error("expected OK=false")
	}
	if resp.Error == "" {
		t.Error("expected error message")
	}

	var data struct {
		Error   string            `json:"error"`
		Matches []ipc.PageSession `json:"matches"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if len(data.Matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(data.Matches))
	}
	if data.Error != resp.Error {
		t.Errorf("data.Error = %q, want %q", data.Error, resp.Error)
	}
}

func TestHandleTabSwitch_EmptyQuery(t *testing.T) {
	d := New(DefaultConfig())
	d.sessions.Add("sess1", "target1", "http://example.com", "Example")

	resp := d.handleTabSwitch("")
	if resp.OK {
		t.Error("expected error from empty-query switch")
	}
	if !contains(resp.Error, "query is required") {
		t.Errorf("expected 'query is required' error, got %q", resp.Error)
	}
}

func TestHandleTabSwitch_NoMatches(t *testing.T) {
	d := New(DefaultConfig())
	d.sessions.Add("sess1", "target1", "http://example.com", "Example")

	resp := d.handleTabSwitch("nonexistent")
	if resp.OK {
		t.Error("expected error for no matches")
	}
	if resp.Error == "" || !contains(resp.Error, "no tab matches query") {
		t.Errorf("expected 'no tab matches query' error, got %q", resp.Error)
	}
}

func TestHandleTabSwitch_Ambiguous(t *testing.T) {
	d := New(DefaultConfig())
	d.sessions.Add("sess1", "target1", "http://example.com", "Test One")
	d.sessions.Add("sess2", "target2", "http://other.com", "Test Two")

	resp := d.handleTabSwitch("Test")
	if resp.OK {
		t.Error("expected error for ambiguous match")
	}
	if !contains(resp.Error, "ambiguous query") {
		t.Errorf("expected 'ambiguous query' error, got %q", resp.Error)
	}
}

func TestHandleTabClose_NoActive(t *testing.T) {
	d := New(DefaultConfig())
	// No sessions added.
	resp := d.handleTabClose("")
	if resp.OK {
		t.Error("expected error for no active tab")
	}
	if !contains(resp.Error, "no active tab") {
		t.Errorf("expected 'no active tab' error, got %q", resp.Error)
	}
}

func TestHandleTabClose_LastTabGuard(t *testing.T) {
	d := New(DefaultConfig())
	d.sessions.Add("sess1", "target1", "http://example.com", "Example")

	resp := d.handleTabClose("")
	if resp.OK {
		t.Error("expected error refusing to close last tab")
	}
	if !contains(resp.Error, "cannot close the last tab") {
		t.Errorf("expected last-tab guard message, got %q", resp.Error)
	}
}

func TestHandleTabList_NoBrowserSession(t *testing.T) {
	d := New(DefaultConfig())
	// No sessions registered: handleTabList itself returns the empty list
	// (it does not call requireBrowser; handleTab is the gating layer).
	resp := d.handleTabList()
	if !resp.OK {
		t.Errorf("expected OK=true even with empty session list, got error %q", resp.Error)
	}
	var data ipc.TabData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.Sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(data.Sessions))
	}
}
