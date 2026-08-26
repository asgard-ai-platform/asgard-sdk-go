package models

// GenericBotMessage represents a message sent from client to the Edge Server
type GenericBotMessage struct {
	CustomChannelId  string                        `json:"customChannelId"`
	CustomMessageId  string                        `json:"customMessageId"`
	Text             string                        `json:"text,omitempty"`
	Action           PostBackAction                `json:"action"`
	BlobIds          []string                      `json:"blobIds,omitempty"`
	Payload          map[string]interface{}        `json:"payload,omitempty"`
	ToolCallConsents []ToolCallConsentResponseItem `json:"toolCallConsents,omitempty"`
	// InvocationId names a Trigger invocation this channel serves, so the run's
	// terminal settles that invocation's record instead of the caller having to watch
	// for it. Only Dispatch reads it — a Trigger is the only caller that has one — and
	// it is ignored by every other endpoint.
	InvocationId *string `json:"invocationId,omitempty"`
}

// PostBackAction defines the action type for a message
type PostBackAction string

const (
	// PostBackActionNone is an ordinary turn. It also opens a brand-new channel:
	// the first message on an unknown customChannelId creates it, so a fresh
	// conversation never needs a reset.
	PostBackActionNone PostBackAction = "NONE"
	// PostBackActionResetChannel wipes an EXISTING channel and then runs this
	// message as the first turn of the fresh one. It is the client-side "start
	// over" button; prefer BotProviderClient.DeleteChannel followed by a NONE
	// turn, which separates the two steps. It must NOT carry BlobIds: the reset
	// deletes every blob uploaded to the channel before the message is
	// dispatched, so the server rejects that request (400) rather than silently
	// dropping the attachments. Delete → UploadBlob → NONE is the working order.
	PostBackActionResetChannel            PostBackAction = "RESET_CHANNEL"
	PostBackActionResponseToolCallConsent PostBackAction = "RESPONSE_TOOL_CALL_CONSENT"
)

// BufferedMessage represents a message returned from the Edge Server
type BufferedMessage struct {
	MessageId              string           `json:"messageId"`
	ReplyToCustomMessageId string           `json:"replyToCustomMessageId"`
	Text                   string           `json:"text"`
	Payload                interface{}      `json:"payload"`
	IsDebug                bool             `json:"isDebug"`
	Idx                    *int             `json:"idx"`
	Template               *MessageTemplate `json:"template"`
	// ParentToolUseId nests subagent (Task) output under its parent tool_use;
	// empty on the main line. Additive (CLI-driver) — flat clients ignore it.
	ParentToolUseId string `json:"parentToolUseId,omitempty"`
}
