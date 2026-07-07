package ipc

import (
	"encoding/json"
	"log"
	"strings"
)

// MultiElementSeparator is the separator string used between multiple elements
// in text output for observation commands (html, css inline, css computed).
const MultiElementSeparator = "--"

// CommandExecutor executes CLI commands with arguments.
// Returns true if the command was recognized, false otherwise.
// Used by the REPL to execute commands via Cobra.
type CommandExecutor func(args []string) (recognized bool, err error)

// Request represents a command sent from the CLI to the daemon.
type Request struct {
	Cmd    string          `json:"cmd"`
	Target string          `json:"target,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Debug  bool            `json:"debug,omitempty"` // Enable debug output for this request
}

// Response represents a response sent from the daemon to the CLI.
type Response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// StatusData is the response data for the "status" command.
type StatusData struct {
	Running       bool          `json:"running"`
	PID           int           `json:"pid,omitempty"`
	ActiveSession *PageSession  `json:"activeSession,omitempty"`
	Sessions      []PageSession `json:"sessions,omitempty"`
}

// ConsoleFrame is a single call frame from a captured stack trace. It mirrors
// a CDP Runtime.CallFrame and adds Async to mark the frames that begin an
// asynchronous continuation.
type ConsoleFrame struct {
	// Function is the frame's function name. Empty for an anonymous function,
	// which is represented as such rather than dropped.
	Function string `json:"function,omitempty"`
	URL      string `json:"url,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	// Async carries the parent group's description (for example "Promise.then")
	// on the first frame of an asynchronous continuation, marking the boundary
	// between synchronous and asynchronous frames. Empty for synchronous frames.
	Async string `json:"async,omitempty"`
}

// ConsolePreviewProp is one property from a RemoteObject's shallow preview. The
// values are the strings CDP delivers inline, so no per-property round trip is
// needed to record a non-primitive argument.
type ConsolePreviewProp struct {
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`
	Subtype string `json:"subtype,omitempty"`
	Value   string `json:"value,omitempty"`
}

// ConsoleArg is a structured console argument mirroring a CDP RemoteObject. It
// preserves an argument's type and value instead of stringifying it, so a
// non-primitive (an object, array, function) records its description and a
// shallow preview rather than collapsing to null.
type ConsoleArg struct {
	// Type is the RemoteObject type (string, number, boolean, object, ...).
	Type string `json:"type,omitempty"`
	// Subtype refines an object-typed value (array, null, error, regexp, ...).
	Subtype string `json:"subtype,omitempty"`
	// Value is the verbatim primitive value as CDP delivered it. Absent for
	// non-primitives, which carry Description and Preview instead.
	Value json.RawMessage `json:"value,omitempty"`
	// Description is the RemoteObject description for a non-primitive.
	Description string `json:"description,omitempty"`
	// Preview is a shallow property preview for a non-primitive.
	Preview []ConsolePreviewProp `json:"preview,omitempty"`
}

// ConsoleEntry represents a console log entry. It carries entries from both
// Runtime (consoleAPICalled, exceptionThrown) and the Log domain (entryAdded),
// distinguished by Source.
type ConsoleEntry struct {
	// Seq is the buffer-assigned sequence number, a stable identifier for the
	// entry across daemon round-trips. Always present in JSON (0 means the
	// entry was never buffered) because agents address entries by it.
	Seq       uint64 `json:"seq"`
	SessionID string `json:"sessionId,omitempty"`
	// Type is the severity level. Runtime console types pass through; Log-domain
	// levels are mapped onto the same set (verbose maps to debug) so --type
	// filtering works uniformly across both streams.
	Type string `json:"type"`
	Text string `json:"text"`
	// Source labels a Log-domain entry's origin (security, network, deprecation,
	// ...). Empty for Runtime console and exception entries.
	Source string `json:"source,omitempty"`
	// Args holds the structured console arguments. Absent for exception and
	// Log-domain entries.
	Args      []ConsoleArg `json:"args,omitempty"`
	Timestamp int64        `json:"timestamp"`
	// URL, Line, and Column summarize the first captured frame as a convenience
	// locator; Stack holds the full call chain.
	URL    string `json:"url,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
	// Stack is the full ordered call stack, including the asynchronous parent
	// chain, for console API calls and exceptions.
	Stack []ConsoleFrame `json:"stack,omitempty"`
	// ExceptionClass is the thrown value's class name (for example TypeError).
	ExceptionClass string `json:"exceptionClass,omitempty"`
	// ExceptionSubtype refines the thrown value (error, ...).
	ExceptionSubtype string `json:"exceptionSubtype,omitempty"`
	// NetworkRequestID links a Log-domain entry to a network buffer entry. Kept
	// verbatim so a consumer can correlate it to the network stream.
	NetworkRequestID string `json:"networkRequestId,omitempty"`
	// WorkerID identifies the worker that produced a Log-domain entry, if any.
	WorkerID string `json:"workerId,omitempty"`
}

// Console type constants matching CDP Runtime.consoleAPICalled types.
const (
	ConsoleTypeLog     = "log"
	ConsoleTypeDebug   = "debug"
	ConsoleTypeInfo    = "info"
	ConsoleTypeError   = "error"
	ConsoleTypeWarning = "warning"
)

// consoleTypeAliases maps user-friendly aliases to CDP canonical types.
var consoleTypeAliases = map[string]string{
	"warn": ConsoleTypeWarning,
}

// NormalizeConsoleType converts a console type string to its canonical CDP form.
func NormalizeConsoleType(t string) string {
	lower := strings.ToLower(t)
	if canonical, ok := consoleTypeAliases[lower]; ok {
		return canonical
	}
	return lower
}

// DefaultMaxBodySize is the default byte budget for network bodies. The CLI uses
// it as the --max-body-size truncation default for the --detail full text list;
// the daemon uses it as the Network.enable maxPostDataSize inline cap. They are
// deliberately equal so a request body that survives truncation also arrives
// inline without a fallback fetch.
const DefaultMaxBodySize = 102400

// MaxBodySizeUnlimited is the --max-body-size sentinel that disables truncation
// entirely, leaving bodies at full fidelity. It is the unset default for JSON,
// text drill-down, and save, where a complete payload is the point.
const MaxBodySizeUnlimited = -1

// NetworkEntry represents a network request/response entry.
type NetworkEntry struct {
	// Seq is the buffer-assigned sequence number, a stable identifier for the
	// entry across daemon round-trips. Always present in JSON (0 means the
	// entry was never buffered). Redirect hops share a CDP RequestID but each
	// is a separate push, so seq is the unambiguous address RequestID is not.
	Seq             uint64            `json:"seq"`
	SessionID       string            `json:"sessionId,omitempty"`
	RequestID       string            `json:"requestId"`
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Type            string            `json:"type,omitempty"`
	Status          int               `json:"status,omitempty"`
	StatusText      string            `json:"statusText,omitempty"`
	MimeType        string            `json:"mimeType,omitempty"`
	RequestTime     int64             `json:"requestTime"`
	ResponseTime    int64             `json:"responseTime,omitempty"`
	Duration        float64           `json:"duration,omitempty"`
	Size            int64             `json:"size,omitempty"`
	RequestHeaders  map[string]string `json:"requestHeaders,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	RequestBody     string            `json:"requestBody,omitempty"`
	// RequestBodyTruncated reports that --max-body-size cut the request body.
	RequestBodyTruncated bool `json:"requestBodyTruncated,omitempty"`
	// ResponseBody holds the response payload returned by the server.
	ResponseBody string `json:"responseBody,omitempty"`
	// ResponseBodyTruncated reports that --max-body-size cut the response body.
	ResponseBodyTruncated bool `json:"responseBodyTruncated,omitempty"`
	// ResponseBodyPath is the file path of a saved binary response body.
	ResponseBodyPath string `json:"responseBodyPath,omitempty"`
	Failed           bool   `json:"failed"`
	Error            string `json:"error,omitempty"`

	// RemoteIPAddress is the server IP that served the response.
	RemoteIPAddress string `json:"remoteIPAddress,omitempty"`
	// RemotePort is the server port that served the response.
	RemotePort int `json:"remotePort,omitempty"`
	// Protocol is the negotiated network protocol (for example h2, http/1.1, h3).
	Protocol string `json:"protocol,omitempty"`
	// FromDiskCache reports the response was served from the disk cache.
	FromDiskCache bool `json:"fromDiskCache,omitempty"`
	// FromServiceWorker reports the response was served by a service worker.
	FromServiceWorker bool `json:"fromServiceWorker,omitempty"`
	// FromPrefetchCache reports the response was served from the prefetch cache.
	FromPrefetchCache bool `json:"fromPrefetchCache,omitempty"`
	// ConnectionID identifies the physical connection that served the response,
	// so requests sharing a connection can be correlated.
	ConnectionID float64 `json:"connectionId,omitempty"`
	// SecurityState is the transport security posture (secure, insecure, neutral, unknown).
	SecurityState string `json:"securityState,omitempty"`
	// Timing is the per-phase latency breakdown derived from the CDP ResourceTiming.
	Timing *NetworkTiming `json:"timing,omitempty"`
	// Initiator records what caused the request: its type and a single source location.
	Initiator *NetworkInitiator `json:"initiator,omitempty"`

	// awaitingRequestBody marks an entry whose request body was advertised
	// (hasPostData) but omitted from requestWillBeSent, so the daemon is
	// fetching it via Network.getRequestPostData off the read loop. It travels
	// with the entry because the ring buffer stores entries by value and
	// compacts them on removal, leaving no stable external identity to key on;
	// a requestId-keyed side map also cannot distinguish redirect hops that
	// share the id. Excluded from JSON so a query racing an in-flight fetch
	// cannot serialize a transient pending flag.
	awaitingRequestBody bool `json:"-"`
}

// AwaitRequestBody marks the entry as awaiting an out-of-band request-body
// fetch. The flag is unexported on the wire (json:"-") but the daemon, in a
// different package, needs to set and read it, so access goes through these
// methods.
func (e *NetworkEntry) AwaitRequestBody() {
	e.awaitingRequestBody = true
}

// AwaitingRequestBody reports whether the entry is awaiting a request-body fetch.
func (e *NetworkEntry) AwaitingRequestBody() bool {
	return e.awaitingRequestBody
}

// SetRequestBody stores a fetched request body and clears the awaiting marker.
func (e *NetworkEntry) SetRequestBody(body string) {
	e.RequestBody = body
	e.awaitingRequestBody = false
}

// ClearAwaitingRequestBody clears the awaiting marker without storing a body,
// used when the fetch returns no data or fails.
func (e *NetworkEntry) ClearAwaitingRequestBody() {
	e.awaitingRequestBody = false
}

// NetworkTiming is a per-phase latency breakdown of a network request, in
// milliseconds. The daemon derives each phase from the CDP ResourceTiming
// offsets (which are relative to a requestTime baseline) so callers read
// durations directly rather than subtracting offsets. A phase that did not
// occur (for example DNS on a reused connection) is omitted.
type NetworkTiming struct {
	// DNSMs is the DNS resolution time (dnsEnd - dnsStart).
	DNSMs float64 `json:"dnsMs,omitempty"`
	// ConnectMs is the TCP connection setup time, excluding the TLS handshake
	// which is reported separately as TLSMs (sslStart - connectStart when a
	// handshake occurred, otherwise connectEnd - connectStart).
	ConnectMs float64 `json:"connectMs,omitempty"`
	// TLSMs is the TLS handshake time (sslEnd - sslStart).
	TLSMs float64 `json:"tlsMs,omitempty"`
	// SendMs is the request send time (sendEnd - sendStart).
	SendMs float64 `json:"sendMs,omitempty"`
	// WaitMs is the time to first byte, from request sent to response headers
	// received (receiveHeadersEnd - sendEnd).
	WaitMs float64 `json:"waitMs,omitempty"`
}

// NetworkInitiator records why a request was made: its initiator type and a
// single source location. The full Runtime.StackTrace parent chain that CDP
// carries for script initiators is deliberately not stored.
type NetworkInitiator struct {
	// Type is the initiator category (parser, script, preload, ...).
	Type string `json:"type,omitempty"`
	// URL is the source location that issued the request, taken from the CDP
	// Initiator's own url or, for script initiators, its top stack frame.
	URL string `json:"url,omitempty"`
	// Line is the 0-based line number within URL.
	Line int `json:"line,omitempty"`
}

// ConsoleData is the response data for the "console" command.
type ConsoleData struct {
	Entries []ConsoleEntry `json:"entries"`
	Count   int            `json:"count"`
}

// NetworkData is the response data for the "network" command.
type NetworkData struct {
	Entries []NetworkEntry `json:"entries"`
	Count   int            `json:"count"`
}

// PageSession represents an active CDP page session.
type PageSession struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Active bool   `json:"active,omitempty"`
	Status int    `json:"status,omitempty"` // HTTP status of last document load
}

// TabParams represents parameters for the "tab" command.
type TabParams struct {
	Action string `json:"action"` // "list", "switch", "new", or "close"
	Query  string `json:"query,omitempty"`
	URL    string `json:"url,omitempty"` // Optional URL for "new"
}

// TabData is the response data for "tab" list and switch/close actions.
type TabData struct {
	ActiveSession string        `json:"activeSession,omitempty"`
	Sessions      []PageSession `json:"sessions"`
}

// NewTabData is the response data for "tab new".
type NewTabData struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// ScreenshotParams represents parameters for the "screenshot" command.
type ScreenshotParams struct {
	FullPage bool `json:"fullPage"`
}

// ScreenshotData is the response data for the "screenshot" command.
type ScreenshotData struct {
	Data []byte `json:"data"`
}

// HTMLParams represents parameters for the "html" command.
type HTMLParams struct {
	Selector string `json:"selector,omitempty"`
}

// ElementWithHTML combines element metadata with HTML
type ElementWithHTML struct {
	ElementMeta
	HTML string `json:"html"`
}

// HTMLData is the response data for the "html" command.
type HTMLData struct {
	HTML      string            `json:"html,omitempty"`      // single result or legacy
	HTMLMulti []ElementWithHTML `json:"htmlMulti,omitempty"` // multi-element with metadata
}

// NavigateParams represents parameters for the "navigate" command.
type NavigateParams struct {
	URL     string `json:"url"`
	Wait    bool   `json:"wait"`    // wait for page load completion
	Timeout int    `json:"timeout"` // timeout in seconds (when wait=true)
}

// NavigateData is the response data for the "navigate" command.
type NavigateData struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ReloadParams represents parameters for the "reload" command.
type ReloadParams struct {
	IgnoreCache bool `json:"ignoreCache"`
	Wait        bool `json:"wait"`    // wait for page load completion
	Timeout     int  `json:"timeout"` // timeout in seconds (when wait=true)
}

// HistoryParams represents parameters for the "back" and "forward" commands.
type HistoryParams struct {
	Wait    bool `json:"wait"`    // wait for page load completion
	Timeout int  `json:"timeout"` // timeout in seconds (when wait=true)
}

// ReadyParams represents parameters for the "ready" command.
type ReadyParams struct {
	Timeout     int    `json:"timeout"`     // timeout in seconds
	Selector    string `json:"selector"`    // CSS selector to wait for (optional)
	NetworkIdle bool   `json:"networkIdle"` // wait for network idle
	Eval        string `json:"eval"`        // JavaScript expression to evaluate (optional)
}

// ClickParams represents parameters for the "click" command.
type ClickParams struct {
	Selector string `json:"selector"`
}

// FocusParams represents parameters for the "focus" command.
type FocusParams struct {
	Selector string `json:"selector"`
}

// TypeParams represents parameters for the "type" command.
type TypeParams struct {
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text"`
	Key      string `json:"key,omitempty"`
	Clear    bool   `json:"clear,omitempty"`
}

// KeyParams represents parameters for the "key" command.
type KeyParams struct {
	Key   string `json:"key"`
	Ctrl  bool   `json:"ctrl,omitempty"`
	Alt   bool   `json:"alt,omitempty"`
	Shift bool   `json:"shift,omitempty"`
	Meta  bool   `json:"meta,omitempty"`
}

// SelectParams represents parameters for the "select" command.
type SelectParams struct {
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

// ScrollParams represents parameters for the "scroll" command.
type ScrollParams struct {
	Selector string `json:"selector,omitempty"`
	ToX      int    `json:"toX,omitempty"`
	ToY      int    `json:"toY,omitempty"`
	ByX      int    `json:"byX,omitempty"`
	ByY      int    `json:"byY,omitempty"`
	Mode     string `json:"mode"` // "element", "to", or "by"
}

// EvalParams represents parameters for the "eval" command.
type EvalParams struct {
	Expression string `json:"expression"`
	Timeout    int    `json:"timeout,omitempty"` // timeout in seconds
}

// EvalData is the response data for the "eval" command.
type EvalData struct {
	Value    any  `json:"value,omitempty"`
	HasValue bool `json:"hasValue,omitempty"`
}

// CookiesParams represents parameters for the "cookies" command.
type CookiesParams struct {
	Action   string `json:"action"` // "list", "set", or "delete"
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	MaxAge   int    `json:"maxAge,omitempty"`   // seconds
	SameSite string `json:"sameSite,omitempty"` // "Strict", "Lax", or "None"
}

// Cookie represents a browser cookie with all CDP attributes.
type Cookie struct {
	Name         string  `json:"name"`
	Value        string  `json:"value"`
	Domain       string  `json:"domain"`
	Path         string  `json:"path"`
	Expires      float64 `json:"expires"`
	Size         int     `json:"size"`
	HTTPOnly     bool    `json:"httpOnly"`
	Secure       bool    `json:"secure"`
	Session      bool    `json:"session"`
	SameSite     string  `json:"sameSite,omitempty"`
	Priority     string  `json:"priority,omitempty"`
	SameParty    bool    `json:"sameParty,omitempty"`
	SourceScheme string  `json:"sourceScheme,omitempty"`
	SourcePort   int     `json:"sourcePort,omitempty"`
}

// CookiesData is the response data for the "cookies" command.
type CookiesData struct {
	Cookies []Cookie `json:"cookies,omitempty"`
	Count   int      `json:"count,omitempty"`
	// For ambiguous delete errors
	Matches []Cookie `json:"matches,omitempty"`
}

// CSSParams represents parameters for the "css" command.
type CSSParams struct {
	Action   string `json:"action"`             // "save", "computed", "get", "inline", or "matched"
	Selector string `json:"selector,omitempty"` // CSS selector for computed/get/inline/matched
	Property string `json:"property,omitempty"` // CSS property for get action
}

// ElementMeta contains element identification metadata extracted from DOM elements.
// The identification follows CSS selector notation for developer familiarity.
//
// Identification Priority:
//  1. ID attribute (if present) -> #id
//  2. First class name (if present) -> .class:N
//  3. Tag name (always present) -> tag:N
//
// Note: Only the first class is captured when an element has multiple classes.
// Special characters in IDs/classes are sanitized to valid CSS identifier characters.
type ElementMeta struct {
	Tag   string `json:"tag"`             // lowercase tag name (div, span, svg, etc.)
	ID    string `json:"id,omitempty"`    // id attribute value (sanitized, if present)
	Class string `json:"class,omitempty"` // first class name only (sanitized, if present)
}

// ElementWithStyles combines element metadata with styles
type ElementWithStyles struct {
	ElementMeta
	Styles map[string]string `json:"styles,omitempty"` // for computed
	Inline string            `json:"inline,omitempty"` // for inline
}

// CSSData is the response data for the "css" command.
type CSSData struct {
	CSS           string              `json:"css,omitempty"`           // For save/matched actions
	Styles        map[string]string   `json:"styles,omitempty"`        // For computed action (single element, JSON format)
	ComputedMulti []ElementWithStyles `json:"computedMulti,omitempty"` // For computed action (multiple elements with metadata)
	Value         string              `json:"value,omitempty"`         // For get action
	InlineMulti   []ElementWithStyles `json:"inlineMulti,omitempty"`   // For inline action (with metadata)
	Inline        []string            `json:"inline,omitempty"`        // Deprecated: For inline action (style attributes only)
	Matched       []CSSMatchedRule    `json:"matched,omitempty"`       // For matched action
}

// CSSMatchedRule represents a CSS rule matched to an element.
type CSSMatchedRule struct {
	Selector   string            `json:"selector"`
	Properties map[string]string `json:"properties"`
	Source     string            `json:"source,omitempty"` // stylesheet URL or "inline"
}

// ServeParams represents parameters for the "serve" command.
type ServeParams struct {
	Action      string   `json:"action"`                // "start" or "stop"
	Mode        string   `json:"mode,omitempty"`        // "static" or "proxy"
	Directory   string   `json:"directory,omitempty"`   // Directory to serve (static mode)
	ProxyURL    string   `json:"proxyURL,omitempty"`    // Backend URL to proxy (proxy mode)
	Port        int      `json:"port,omitempty"`        // Server port (0 = auto-detect)
	Host        string   `json:"host,omitempty"`        // Bind host ("localhost" or "0.0.0.0")
	WatchPaths  []string `json:"watchPaths,omitempty"`  // Paths to watch for changes
	IgnorePaths []string `json:"ignorePaths,omitempty"` // Glob patterns to ignore
}

// ServeData is the response data for the "serve" command.
type ServeData struct {
	Running bool   `json:"running"`
	Mode    string `json:"mode,omitempty"`
	URL     string `json:"url,omitempty"`
	Port    int    `json:"port,omitempty"`
}

// SuccessResponse creates a successful response with the given data.
func SuccessResponse(data any) Response {
	var raw json.RawMessage
	if data != nil {
		var err error
		raw, err = json.Marshal(data)
		if err != nil {
			log.Printf("ipc: failed to marshal response data: %v", err)
			return ErrorResponse("internal error: failed to marshal response")
		}
	}
	return Response{OK: true, Data: raw}
}

// ErrorResponse creates an error response with the given message.
func ErrorResponse(msg string) Response {
	return Response{OK: false, Error: msg}
}
