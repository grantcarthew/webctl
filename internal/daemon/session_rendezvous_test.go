package daemon

import (
	"testing"
	"time"
)

func TestSessionManager_NetworkEnableClaimAtMostOnce(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("s1", "t1", "", "")

	if sm.NetworkEnabled("s1") {
		t.Error("session should start with Network disabled")
	}
	if !sm.ClaimNetworkEnable("s1") {
		t.Error("first claim should win")
	}
	if sm.ClaimNetworkEnable("s1") {
		t.Error("second claim should lose (at-most-once)")
	}
	if !sm.NetworkEnabled("s1") {
		t.Error("Network should read as enabled after a winning claim")
	}

	// Clear allows a retry, mirroring a failed Network.enable.
	sm.ClearNetworkEnabled("s1")
	if sm.NetworkEnabled("s1") {
		t.Error("Network should read as disabled after clear")
	}
	if !sm.ClaimNetworkEnable("s1") {
		t.Error("claim should win again after clear")
	}

	if sm.ClaimNetworkEnable("missing") {
		t.Error("claim on an unknown session should return false")
	}
}

func TestSessionManager_WaitForAttach_FastPath(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("s1", "t1", "http://u", "Title")

	sess, ch := sm.waitForAttach("t1")
	if sess == nil {
		t.Fatal("expected fast-path session for an already-attached target")
	}
	if ch != nil {
		t.Error("expected nil channel on the fast path")
	}
	if sess.ID != "s1" {
		t.Errorf("session ID = %q, want s1", sess.ID)
	}
}

func TestSessionManager_WaitForAttach_SignalledByAdd(t *testing.T) {
	sm := NewSessionManager()

	sess, ch := sm.waitForAttach("t1")
	if sess != nil || ch == nil {
		t.Fatal("expected a waiter channel for an unattached target")
	}

	go sm.Add("s1", "t1", "http://u", "Title")

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("Add did not signal the attach waiter")
	}
	if sm.GetByTargetID("t1") == nil {
		t.Error("session should be present after Add")
	}
}

func TestSessionManager_WaitForDetach_FastPathWhenAbsent(t *testing.T) {
	sm := NewSessionManager()
	if ch := sm.waitForDetach("missing"); ch != nil {
		t.Error("expected nil channel (fast path) when the session is already gone")
	}
}

func TestSessionManager_WaitForDetach_SignalledByRemove(t *testing.T) {
	sm := NewSessionManager()
	sm.Add("s1", "t1", "http://u", "Title")
	sm.Add("s2", "t2", "http://u2", "Title2")

	ch := sm.waitForDetach("s1")
	if ch == nil {
		t.Fatal("expected a waiter channel for a present session")
	}

	go sm.Remove("s1")

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("Remove did not signal the detach waiter")
	}
}
