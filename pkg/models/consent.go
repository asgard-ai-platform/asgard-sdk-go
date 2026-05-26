package models

// PendingToolCall describes a single tool invocation awaiting user consent.
type PendingToolCall struct {
	ToolCallId     string      `json:"toolCallId"`
	ToolsetName    string      `json:"toolsetName"` // empty for builtin tools
	ToolName       string      `json:"toolName"`
	Parameter      interface{} `json:"parameter"`
	// Reason is the short user-facing rationale the agent supplied via the
	// `_reason` meta property on the tool input. Empty when not supplied.
	Reason         string      `json:"reason"`
	AlreadyAllowed bool        `json:"alreadyAllowed"`
}

// ToolCallConsentResponseItem is the per-tool decision provided by the user
// when sending action=RESPONSE_TOOL_CALL_CONSENT.
type ToolCallConsentResponseItem struct {
	ToolCallId string                `json:"toolCallId"`
	Result     ToolCallConsentResult `json:"result"`
	DenyReason string                `json:"denyReason,omitempty"`
}

// ToolCallConsentResult is the decision value for a single tool call consent.
type ToolCallConsentResult string

const (
	ToolCallConsentResultAllowOnce   ToolCallConsentResult = "ALLOW_ONCE"
	ToolCallConsentResultAllowAlways ToolCallConsentResult = "ALLOW_ALWAYS"
	ToolCallConsentResultDenyOnce    ToolCallConsentResult = "DENY_ONCE"
)
