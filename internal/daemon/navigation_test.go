package daemon

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grantcarthew/webctl/internal/cdp"
)

// isClosed reports whether a milestone channel has been closed without blocking.
func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestNavigation_DOMReadyDoesNotImplyLoaded(t *testing.T) {
	n := newNavigation()
	if isClosed(n.DOMReady()) {
		t.Fatal("DOMReady closed before markDOMReady")
	}
	n.markDOMReady()
	if !isClosed(n.DOMReady()) {
		t.Error("DOMReady not closed after markDOMReady")
	}
	if isClosed(n.Loaded()) {
		t.Error("Loaded should not close on markDOMReady")
	}
}

func TestNavigation_LoadImpliesDOMReady(t *testing.T) {
	n := newNavigation()
	n.markLoaded()
	if !isClosed(n.Loaded()) {
		t.Error("Loaded not closed after markLoaded")
	}
	if !isClosed(n.DOMReady()) {
		t.Error("DOMReady not closed by markLoaded (load implies DOM-ready)")
	}
}

func TestNavigation_IdempotentMarksFirstValueWins(t *testing.T) {
	n := newNavigation()
	// Repeated marks must not panic on a double close.
	n.markDOMReady()
	n.markDOMReady()
	n.markLoaded()
	n.markLoaded()
	n.markFrameNavigated()
	n.markFrameNavigated()
	n.cancel(cancelSuperseded)
	n.cancel(cancelDetached)

	if got := n.CancelReason(); got != cancelSuperseded {
		t.Errorf("CancelReason = %v, want cancelSuperseded (first reason wins)", got)
	}
}

func TestNavigation_FrameNavigatedCloses(t *testing.T) {
	n := newNavigation()
	if isClosed(n.FrameNavigated()) {
		t.Fatal("FrameNavigated closed before markFrameNavigated")
	}
	n.markFrameNavigated()
	if !isClosed(n.FrameNavigated()) {
		t.Error("FrameNavigated not closed after markFrameNavigated")
	}
}

func TestAwaitMilestone_ReachedBeforeAwaitReturnsPromptly(t *testing.T) {
	n := newNavigation()
	n.markLoaded() // milestone reached before anyone awaits it

	got := awaitMilestone(n.Loaded(), n.Cancelled(), time.Second)
	if got != navReached {
		t.Errorf("awaitMilestone = %v, want navReached for an already-reached milestone", got)
	}
}

func TestAwaitMilestone_MilestoneWinsWhenBothClosed(t *testing.T) {
	n := newNavigation()
	n.markLoaded()             // navigation succeeded
	n.cancel(cancelSuperseded) // then a later navigation superseded it

	// Both Loaded and Cancelled are closed; the milestone must win deterministically.
	for i := 0; i < 100; i++ {
		if got := awaitMilestone(n.Loaded(), n.Cancelled(), time.Second); got != navReached {
			t.Fatalf("awaitMilestone = %v, want navReached when the milestone closed before cancellation", got)
		}
	}
}

func TestAwaitMilestone_Timeout(t *testing.T) {
	n := newNavigation()
	got := awaitMilestone(n.Loaded(), n.Cancelled(), 10*time.Millisecond)
	if got != navTimedOut {
		t.Errorf("awaitMilestone = %v, want navTimedOut", got)
	}
}

func TestNavTracker_BeginSupersedesPrior(t *testing.T) {
	tr := newNavTracker()
	a := tr.begin("s")
	b := tr.begin("s")

	if !isClosed(a.Cancelled()) {
		t.Error("prior navigation was not cancelled by begin")
	}
	if a.CancelReason() != cancelSuperseded {
		t.Errorf("prior cancel reason = %v, want cancelSuperseded", a.CancelReason())
	}
	if isClosed(b.Cancelled()) {
		t.Error("replacement navigation should not be cancelled")
	}
	if tr.current("s") != b {
		t.Error("current should be the replacement navigation")
	}
}

func TestNavTracker_ClearCancelsWithDetach(t *testing.T) {
	tr := newNavTracker()
	n := tr.begin("s")
	tr.clear("s")

	if !isClosed(n.Cancelled()) {
		t.Error("clear did not cancel the navigation")
	}
	if n.CancelReason() != cancelDetached {
		t.Errorf("cancel reason = %v, want cancelDetached", n.CancelReason())
	}
	if tr.current("s") != nil {
		t.Error("current should be nil after clear")
	}
}

// A --wait consumer must return the explicit superseded error (behavioral
// contract item 7) when its navigation is cancelled by a supersession, rather
// than waiting out its timeout.
func TestWaitConsumer_SupersededYieldsSupersededError(t *testing.T) {
	tr := newNavTracker()
	nav := tr.begin("s")
	tr.begin("s") // supersede

	// A long timeout still returns promptly via the Cancelled milestone.
	if got := awaitMilestone(nav.Loaded(), nav.Cancelled(), 5*time.Second); got != navCancelled {
		t.Fatalf("awaitMilestone = %v, want navCancelled", got)
	}

	resp := cancelledNavResponse(nav, "s")
	if resp.OK {
		t.Fatal("expected error response for superseded navigation")
	}
	if resp.Error != errNavigationSuperseded {
		t.Errorf("error = %q, want %q", resp.Error, errNavigationSuperseded)
	}
}

// A detach-reason Cancelled must surface a session-closed error naming the
// session, not the supersession message.
func TestWaitConsumer_DetachYieldsSessionClosedError(t *testing.T) {
	tr := newNavTracker()
	nav := tr.begin("sess-x")
	nav.cancel(cancelDetached)

	resp := cancelledNavResponse(nav, "sess-x")
	if resp.OK {
		t.Fatal("expected error response for detached navigation")
	}
	if resp.Error == errNavigationSuperseded {
		t.Error("detach should not report the supersession message")
	}
	if !strings.Contains(resp.Error, "sess-x") || !strings.Contains(resp.Error, "closed") {
		t.Errorf("error %q should name the closed session", resp.Error)
	}
}

func TestWaitForLoadEvent_NoNavigationReturnsImmediately(t *testing.T) {
	d := New(DefaultConfig())
	if err := d.waitForDOMReady("none", time.Second); err != nil {
		t.Errorf("expected nil when no navigation in flight, got %v", err)
	}
}

func TestWaitForLoadEvent_AlreadyDOMReadyReturnsPromptly(t *testing.T) {
	d := New(DefaultConfig())
	nav := d.navTracker.begin("s1")
	nav.markDOMReady()
	if err := d.waitForDOMReady("s1", time.Second); err != nil {
		t.Errorf("expected prompt nil for already-DOM-ready navigation, got %v", err)
	}
}

// When DOM-ready and a cancellation are both reached, ready reports success: the
// page is ready regardless of a concurrent supersession or detach.
func TestWaitForLoadEvent_DOMReadyWinsOverCancel(t *testing.T) {
	d := New(DefaultConfig())
	nav := d.navTracker.begin("s1")
	nav.markDOMReady()
	nav.cancel(cancelDetached)

	for i := 0; i < 100; i++ {
		if err := d.waitForDOMReady("s1", time.Second); err != nil {
			t.Fatalf("waitForDOMReady = %v, want nil when DOM-ready was reached", err)
		}
	}
}

// ready default mode re-binds to a superseding navigation rather than erroring,
// then returns on the newer navigation's DOM-ready.
func TestWaitForLoadEvent_RebindsOnSupersede(t *testing.T) {
	d := New(DefaultConfig())
	d.navTracker.begin("s1")

	done := make(chan error, 1)
	go func() { done <- d.waitForDOMReady("s1", 5*time.Second) }()

	navB := d.navTracker.begin("s1") // supersede the navigation the consumer bound to
	navB.markDOMReady()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("waitForDOMReady returned %v, want nil after re-bind to newer navigation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waitForDOMReady did not return after re-binding to the superseding navigation")
	}
}

// ready default mode returns when the session detaches, with a session-closed
// error rather than the supersession message.
func TestWaitForLoadEvent_DetachReasonReturnsSessionClosed(t *testing.T) {
	d := New(DefaultConfig())
	nav := d.navTracker.begin("s1")
	nav.cancel(cancelDetached)

	err := d.waitForDOMReady("s1", time.Second)
	if err == nil {
		t.Fatal("expected session-closed error after detach")
	}
	if !strings.Contains(err.Error(), "s1") || !strings.Contains(err.Error(), "closed") {
		t.Errorf("error %q should name the closed session", err)
	}
}

// Driving the real detach producer must cancel the in-flight navigation; this
// fails if the navTracker.clear call site goes missing from handleTargetDetached.
func TestHandleTargetDetached_CancelsInFlightNavigation(t *testing.T) {
	d := New(DefaultConfig())
	nav := d.navTracker.begin("sess-detach")

	params, _ := json.Marshal(map[string]any{"sessionId": "sess-detach"})
	d.handleTargetDetached(cdp.Event{
		Method:    "Target.detachedFromTarget",
		Params:    params,
		SessionID: "sess-detach",
	})

	if !isClosed(nav.Cancelled()) {
		t.Fatal("handleTargetDetached did not cancel the in-flight navigation (missing navTracker.clear)")
	}
	if nav.CancelReason() != cancelDetached {
		t.Errorf("cancel reason = %v, want cancelDetached", nav.CancelReason())
	}
	if d.navTracker.current("sess-detach") != nil {
		t.Error("navigation should be cleared from tracker after detach")
	}
}

// Controlled-event-order proof that ready default mode returns at DOM-ready and
// not at full load: only domContentEventFired is delivered, never loadEventFired.
func TestReadyDefaultMode_ReturnsAtDOMReadyBeforeLoad(t *testing.T) {
	d := New(DefaultConfig())
	d.navTracker.begin("s1")

	done := make(chan error, 1)
	go func() { done <- d.waitForDOMReady("s1", 5*time.Second) }()

	// Deliver ONLY the DOM-ready producer event. loadEventFired is never delivered,
	// so a return here can only come from the DOM-ready milestone.
	d.handleDOMContentEventFired(cdp.Event{SessionID: "s1"})

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("waitForDOMReady returned %v, want nil at DOM-ready", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ready default mode did not return at DOM-ready (no loadEventFired was delivered)")
	}
}
