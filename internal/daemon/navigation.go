package daemon

import "sync"

// cancelReason distinguishes why a Navigation was cancelled, so a woken consumer
// can report the true cause instead of assuming supersession.
type cancelReason int

const (
	// cancelSuperseded means a newer navigation for the same session replaced this one.
	cancelSuperseded cancelReason = iota
	// cancelDetached means the session detached (tab/page closed) while this navigation lived.
	cancelDetached
)

// Navigation represents one in-flight or just-completed navigation for a session.
//
// Each milestone is a broadcast: a channel closed exactly once when the milestone
// is reached. Awaiting an already-reached milestone returns immediately and any
// number of waiters observe it, so "the event already happened" and "I am waiting
// for it" collapse into a single receive. Mark operations are idempotent and safe
// under concurrent calls, which removes the use-after-close hazard by construction.
type Navigation struct {
	mu sync.Mutex

	domReady  chan struct{}
	loaded    chan struct{}
	frameNav  chan struct{}
	cancelled chan struct{}

	domReadyClosed  bool
	loadedClosed    bool
	frameNavClosed  bool
	cancelledClosed bool

	reason cancelReason // cancellation cause; readable after cancelled closes
}

// newNavigation creates a Navigation with all milestones open.
func newNavigation() *Navigation {
	return &Navigation{
		domReady:  make(chan struct{}),
		loaded:    make(chan struct{}),
		frameNav:  make(chan struct{}),
		cancelled: make(chan struct{}),
	}
}

// DOMReady returns the DOM-ready milestone, closed on Page.domContentEventFired
// or Page.loadEventFired (load implies DOM-ready).
func (n *Navigation) DOMReady() <-chan struct{} { return n.domReady }

// Loaded returns the Loaded milestone, closed on Page.loadEventFired.
func (n *Navigation) Loaded() <-chan struct{} { return n.loaded }

// FrameNavigated returns the FrameNavigated milestone, closed on a main-frame
// Page.frameNavigated.
func (n *Navigation) FrameNavigated() <-chan struct{} { return n.frameNav }

// Cancelled returns the Cancelled milestone, closed when this navigation is
// superseded or its session detaches. CancelReason is readable once it closes.
func (n *Navigation) Cancelled() <-chan struct{} { return n.cancelled }

// markDOMReady closes the DOM-ready milestone. Idempotent.
func (n *Navigation) markDOMReady() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closeDOMReadyLocked()
}

func (n *Navigation) closeDOMReadyLocked() {
	if !n.domReadyClosed {
		n.domReadyClosed = true
		close(n.domReady)
	}
}

// markLoaded closes the Loaded milestone and, since load implies DOM-ready, the
// DOM-ready milestone too. Idempotent.
func (n *Navigation) markLoaded() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closeDOMReadyLocked()
	if !n.loadedClosed {
		n.loadedClosed = true
		close(n.loaded)
	}
}

// markFrameNavigated closes the FrameNavigated milestone. The first call wins.
// Idempotent.
func (n *Navigation) markFrameNavigated() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.frameNavClosed {
		n.frameNavClosed = true
		close(n.frameNav)
	}
}

// cancel records the cancellation reason and closes the Cancelled milestone. The
// first reason wins. Idempotent.
func (n *Navigation) cancel(reason cancelReason) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.cancelledClosed {
		n.reason = reason
		n.cancelledClosed = true
		close(n.cancelled)
	}
}

// CancelReason returns why the navigation was cancelled. Call only after the
// Cancelled milestone has closed.
func (n *Navigation) CancelReason() cancelReason {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.reason
}

// navTracker owns the mapping from sessionID to its current Navigation. Read-loop
// producers reach the current navigation through current() and mark milestones;
// request-goroutine consumers begin() a navigation or read the current one and
// await a milestone. Registering interest is creating or reading the Navigation
// and completion is a closed channel, so the register-before-fire race and the
// double-check that the legacy maps required do not exist here.
type navTracker struct {
	mu  sync.Mutex
	nav map[string]*Navigation
}

// newNavTracker creates an empty navigation tracker.
func newNavTracker() *navTracker {
	return &navTracker{nav: make(map[string]*Navigation)}
}

// begin starts a new navigation for the session, cancelling and replacing any
// prior navigation with the supersession reason. The replacement is stored before
// the lock is released, so a consumer woken by the prior navigation's Cancelled
// observes the new navigation when it re-reads current().
func (t *navTracker) begin(sessionID string) *Navigation {
	t.mu.Lock()
	defer t.mu.Unlock()
	if prev, ok := t.nav[sessionID]; ok {
		prev.cancel(cancelSuperseded)
	}
	n := newNavigation()
	t.nav[sessionID] = n
	return n
}

// current returns the session's current navigation, or nil if none is tracked.
func (t *navTracker) current(sessionID string) *Navigation {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.nav[sessionID]
}

// clear removes the session's navigation, cancelling it with the detach reason so
// a blocked consumer wakes with the session-closed outcome rather than waiting out
// its timeout. Safe to call when no navigation is tracked.
func (t *navTracker) clear(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n, ok := t.nav[sessionID]; ok {
		n.cancel(cancelDetached)
		delete(t.nav, sessionID)
	}
}
