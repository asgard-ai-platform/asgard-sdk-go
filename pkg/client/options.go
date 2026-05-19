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
}
