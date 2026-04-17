package client

// MessageRequestOptions holds per-request configuration for SendMessage and NewStreamer.
// A nil value is treated as zero (no debug, no identity hint).
type MessageRequestOptions struct {
	// IsDebug enables debug mode for the request (appends ?is_debug=true).
	IsDebug bool

	// UserIdentityHint is forwarded as the X-ASGARD-USER-IDENTITY-HINT header.
	// Use this to pass a caller-supplied user identity string (max 128 chars).
	UserIdentityHint string
}
