package client

// MessageRequestOptions holds per-request configuration for SendMessage and NewStreaming.
// A nil value is treated as zero (no debug, no identity hint, no consent bypass).
type MessageRequestOptions struct {
	// IsDebug enables debug mode for the request. Appended as ?is_debug=true on both
	// the REST (SendMessage) and SSE (NewStreaming) endpoints.
	IsDebug bool

	// UserIdentityHint is forwarded as the X-ASGARD-USER-IDENTITY-HINT header.
	// Use this to pass a caller-supplied user identity string (max 128 chars).
	UserIdentityHint string

	// BypassToolCallConsent treats every tool call in this single request as
	// already consented. The server skips the consent gate (no
	// asgard.tool_call.consent event, no pause) for this run only — the
	// persistent tool_call_allow_list is not modified. Only honored by the SSE
	// endpoint (NewStreaming) today; the REST endpoint will ignore it
	// server-side.
	BypassToolCallConsent bool

	// LastEventID resumes an in-flight turn instead of dispatching a new one.
	// When set, NewStreamer sends it as the SSE Last-Event-ID request header, so
	// the server performs a pure resubscribe from that cursor and does NOT
	// re-dispatch the message (the request body's CustomChannelId is still used).
	// Leave empty for a fresh send.
	//
	// This is primarily for backend relays that forward a downstream client's
	// (e.g. a browser's) reconnect: the relay MUST propagate the inbound
	// Last-Event-ID here, otherwise the cursor is lost at the SDK boundary and
	// the server treats the reconnect as a fresh send — re-dispatching the turn
	// (a duplicate run). For most in-process callers this stays empty: the
	// streamer resumes network drops automatically without any cursor handling.
	LastEventID string
}

// ChannelStreamOptions holds per-request configuration for NewChannelStreamer
// (the GET rejoin stream). A nil value is treated as zero.
type ChannelStreamOptions struct {
	// LastEventID resumes the rejoin from a durable transcript-seq cursor (the
	// value a prior streamer reported via LastEventID(), or a downstream client's
	// Last-Event-ID). Sent as the SSE Last-Event-ID request header. Empty replays
	// the full collapsed history, then streams the in-flight turn.
	//
	// Propagate a downstream client's inbound Last-Event-ID here when relaying, so
	// the resume cursor survives the SDK boundary.
	LastEventID string

	// UserIdentityHint is forwarded as the X-ASGARD-USER-IDENTITY-HINT header.
	UserIdentityHint string
}

// SuspendOptions holds per-request configuration for SuspendChannel. A nil
// value is treated as zero, which stops whichever run is currently active and
// lets it stop gracefully — what a plain "stop what you're doing" means.
type SuspendOptions struct {
	// RequestID pins the suspend to one specific run, so a caller that has been
	// tracking a run cannot accidentally stop a newer one that replaced it while
	// it was not looking. Naming a run that is no longer current is a no-op
	// success. Empty targets the active run.
	RequestID string

	// Force abandons the run immediately instead of letting it stop gracefully.
	// Reach for it only when a graceful suspend has not taken effect, since
	// in-flight work is dropped rather than allowed to wind down — normally the
	// run should be given the chance to stop cleanly so the conversation stays
	// resumable.
	Force bool

	// UserIdentityHint is forwarded as the X-ASGARD-USER-IDENTITY-HINT header.
	UserIdentityHint string
}

// FeedbackOptions holds per-request configuration for SendMessageFeedback. A
// nil value is treated as zero (no identity hint — the server records the
// rating as "primary").
type FeedbackOptions struct {
	// UserIdentityHint is forwarded as the X-ASGARD-USER-IDENTITY-HINT header —
	// who rated the reply. Pass it whenever the caller knows the end user (a
	// relay with an authenticated user id), since a rating without an identity
	// is much less useful to analyze.
	UserIdentityHint string
}
