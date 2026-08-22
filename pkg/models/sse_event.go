package models

// GenericBotSseEvent represents a Server-Sent Event from the Edge Server
type GenericBotSseEvent struct {
	EventType       SseEventType           `json:"eventType"`
	RequestId       string                 `json:"requestId"`
	EventId         string                 `json:"eventId"`
	Namespace       string                 `json:"namespace"`
	BotProviderName string                 `json:"botProviderName"`
	CustomChannelId string                 `json:"customChannelId"`
	Fact            GenericBotSseEventFact `json:"fact"`
}

// GenericBotSseEventFact contains the polymorphic event data
// Only one field will be non-nil depending on the EventType
type GenericBotSseEventFact struct {
	RunInit         *GenericBotSseEventFactRunInit         `json:"runInit"`
	RunDone         *GenericBotSseEventFactRunDone         `json:"runDone"`
	RunError        *GenericBotSseEventFactRunError        `json:"runError"`
	ProcessStart    *GenericBotSseEventFactProcessStart    `json:"processStart"`
	ProcessComplete *GenericBotSseEventFactProcessComplete `json:"processComplete"`
	MessageStart    *GenericBotSseEventFactMessage         `json:"messageStart"`
	MessageDelta    *GenericBotSseEventFactMessage         `json:"messageDelta"`
	MessageComplete *GenericBotSseEventFactMessage         `json:"messageComplete"`
	// Thinking facts are additive (CLI-driver): extended-thinking blocks stream and
	// complete separately from the assistant message, reusing the message fact shape.
	MessageThinkingStart    *GenericBotSseEventFactMessage `json:"messageThinkingStart"`
	MessageThinkingDelta    *GenericBotSseEventFactMessage `json:"messageThinkingDelta"`
	MessageThinkingComplete *GenericBotSseEventFactMessage `json:"messageThinkingComplete"`
	// Canvas facts (additive) reuse the message fact shape: a `delta` carries the
	// markup that became available in the message's `text`; the `complete` carries
	// the whole fragment in the message's `template` (type CANVAS), and is
	// authoritative. See the SseEventTypeMessageCanvas* constants.
	MessageCanvasStart    *GenericBotSseEventFactMessage              `json:"messageCanvasStart"`
	MessageCanvasDelta    *GenericBotSseEventFactMessage              `json:"messageCanvasDelta"`
	MessageCanvasComplete *GenericBotSseEventFactMessage              `json:"messageCanvasComplete"`
	ToolCallStart         *GenericBotSseEventFactToolCallStart        `json:"toolCallStart"`
	ToolCallComplete      *GenericBotSseEventFactToolCallComplete     `json:"toolCallComplete"`
	ToolCallConsent       *GenericBotSseEventFactToolCallConsent      `json:"toolCallConsent"`
	CompletionModelUsage  *GenericBotSseEventFactCompletionModelUsage `json:"completionModelUsage"`
	SandboxLaunch         *GenericBotSseEventFactSandboxLaunch        `json:"sandboxLaunch"`
	SandboxReady          *GenericBotSseEventFactSandboxReady         `json:"sandboxReady"`
	// Subagent lifecycle facts (additive): started/completed for a subagent the
	// agent spawned. Correlate by AgentId (or by ParentToolUseId, which also tags
	// the subagent's own message/tool_call events) to maintain a live list.
	SubagentStart    *GenericBotSseEventFactSubagentStart    `json:"subagentStart"`
	SubagentComplete *GenericBotSseEventFactSubagentComplete `json:"subagentComplete"`
	// MessageUser (additive) is the user's own turn, replayed when rejoining a
	// channel's history so the user side of the conversation can be rendered.
	// ChannelTitleUpdate (additive) signals the conversation title changed.
	// ChannelStatusUpdate (additive) signals the agent declared where the work
	// stands. Clients that don't use them can ignore these fields.
	MessageUser         *GenericBotSseEventFactMessageUser         `json:"messageUser"`
	ChannelTitleUpdate  *GenericBotSseEventFactChannelTitleUpdate  `json:"channelTitleUpdate"`
	ChannelStatusUpdate *GenericBotSseEventFactChannelStatusUpdate `json:"channelStatusUpdate"`
	// PromptSuggestion (additive) is a prediction of the user's next message,
	// offered as input-box placeholder text.
	PromptSuggestion *GenericBotSseEventFactPromptSuggestion `json:"promptSuggestion"`
}

// GenericBotSseEventFactRunInit is emitted when a run initializes
type GenericBotSseEventFactRunInit struct{}

// GenericBotSseEventFactRunDone is emitted when a run completes successfully
type GenericBotSseEventFactRunDone struct{}

// GenericBotSseEventFactRunError is emitted when a run encounters an error
type GenericBotSseEventFactRunError struct {
	Error ErrorDetail `json:"error"`
}

// GenericBotSseEventFactProcessStart is emitted when a process starts
type GenericBotSseEventFactProcessStart struct {
	ProcessId string       `json:"processId"`
	Task      *interface{} `json:"task"`
}

// GenericBotSseEventFactProcessComplete is emitted when a process completes
type GenericBotSseEventFactProcessComplete struct {
	ProcessId  string       `json:"processId"`
	TaskResult *interface{} `json:"taskResult"`
}

// GenericBotSseEventFactMessage is emitted for message-related events
type GenericBotSseEventFactMessage struct {
	Message BufferedMessage `json:"message"`
}

// GenericBotSseEventFactToolCallStart is emitted when a tool call starts
type GenericBotSseEventFactToolCallStart struct {
	ProcessId string   `json:"processId"`
	CallSeq   int      `json:"callSeq"`
	ToolCall  ToolCall `json:"toolCall"`
	// ToolUseId / ParentToolUseId are additive (CLI-driver): the CLI correlates
	// start↔complete by tool_use id, and nests subagent tool calls under a parent
	// Task tool_use. Empty for pre-CLI-driver servers.
	ToolUseId       string `json:"toolUseId,omitempty"`
	ParentToolUseId string `json:"parentToolUseId,omitempty"`
}

// GenericBotSseEventFactToolCallComplete is emitted when a tool call completes
type GenericBotSseEventFactToolCallComplete struct {
	ProcessId      string      `json:"processId"`
	CallSeq        int         `json:"callSeq"`
	ToolCall       ToolCall    `json:"toolCall"`
	ToolCallResult interface{} `json:"toolCallResult"`
	// ToolUseResultSidecar is an optional structured companion to ToolCallResult:
	// arbitrary JSON a tool may return in addition to the human-readable result
	// string, forwarded verbatim. It lets clients consume a tool's structured
	// output — e.g. a stable id to correlate a multi-step tool's results (build or
	// track a list by id) — without parsing the result string. nil when the tool
	// returned no structured data.
	ToolUseResultSidecar interface{} `json:"toolUseResultSidecar,omitempty"`
	// Additive (CLI-driver): correlation id + subagent nesting + error flag.
	ToolUseId       string `json:"toolUseId,omitempty"`
	ParentToolUseId string `json:"parentToolUseId,omitempty"`
	IsError         bool   `json:"isError,omitempty"`
}

// GenericBotSseEventFactToolCallConsent is emitted when the bot requires user consent for pending tool calls
type GenericBotSseEventFactToolCallConsent struct {
	ProcessId    string            `json:"processId"`
	PendingCalls []PendingToolCall `json:"pendingCalls"`
}

// GenericBotSseEventFactCompletionModelUsage is emitted when a completion model reports token usage
type GenericBotSseEventFactCompletionModelUsage struct {
	ProcessId           string `json:"processId"`
	CompletionModelName string `json:"completionModelName"`
	InputTokens         int64  `json:"inputTokens"`
	OutputTokens        int64  `json:"outputTokens"`
	TotalTokens         int64  `json:"totalTokens"`
	// IsPreset reports whether this usage came from a platform-provided preset
	// completion model, so billing can meter it apart from usage on your own
	// model key.
	IsPreset bool `json:"isPreset"`
}

// GenericBotSseEventFactSandboxLaunch is emitted when a sandbox starts launching
type GenericBotSseEventFactSandboxLaunch struct {
	SandboxName   string `json:"sandboxName"`
	BlueprintName string `json:"blueprintName"`
}

// GenericBotSseEventFactSandboxReady is emitted when a sandbox is ready
type GenericBotSseEventFactSandboxReady struct {
	SandboxName   string `json:"sandboxName"`
	BlueprintName string `json:"blueprintName"`
}

// GenericBotSseEventFactSubagentStart is emitted when a spawned subagent begins
// running. AgentId identifies the subagent; ParentToolUseId is the id of the tool
// call that spawned it — the same value tags the subagent's own message and
// tool_call events, so a client can group its activity and maintain a live list of
// running subagents keyed by AgentId. SubagentType / Description describe the work.
type GenericBotSseEventFactSubagentStart struct {
	AgentId         string `json:"agentId"`
	ParentToolUseId string `json:"parentToolUseId"`
	SubagentType    string `json:"subagentType,omitempty"`
	Description     string `json:"description,omitempty"`
}

// GenericBotSseEventFactSubagentComplete is emitted when a spawned subagent
// finishes. AgentId / ParentToolUseId match the corresponding SubagentStart.
// Status is the terminal outcome ("completed", "failed", or "cancelled"); Summary is the subagent's
// final report.
type GenericBotSseEventFactSubagentComplete struct {
	AgentId         string `json:"agentId"`
	ParentToolUseId string `json:"parentToolUseId"`
	SubagentType    string `json:"subagentType,omitempty"`
	Status          string `json:"status"`
	Summary         string `json:"summary,omitempty"`
}

// GenericBotSseEventFactMessageUser is the user's own turn, surfaced when
// replaying a channel's history so a client can render the user side of the
// conversation. Text is the message the user sent; IdentityHint identifies which
// end user sent it (a caller-supplied hint, "primary" when unspecified); BlobIds
// are the ids of any files the turn attached. Delivered only on a history rejoin.
//
// Blobs (additive) is those same attachments with the metadata needed to RENDER
// them — see MessageBlob. BlobIds stays populated beside it, so code written
// against the older shape is unaffected.
//
// Blobs is EMPTY on turns recorded before the platform started carrying it, and
// there is no backfill — so a client must still handle ids-without-metadata
// rather than treating an id it cannot render as nothing at all. That failure is
// what the field exists to fix: with ids alone there is no name to label a chip
// and no way to tell an image from a document, so an attachment-only turn (empty
// Text) drew nothing whatsoever and the user's own upload went missing from their
// history. Render a neutral attachment chip in that case.
//
// It carries no download URL by design. A presigned link expires — often while
// the page is still open — so it cannot belong to a durable transcript; a client
// that wants the bytes asks for a fresh link at render time.
type GenericBotSseEventFactMessageUser struct {
	MessageId       string        `json:"messageId"`
	Text            string        `json:"text"`
	IdentityHint    string        `json:"identityHint,omitempty"`
	CustomMessageId string        `json:"customMessageId,omitempty"`
	BlobIds         []string      `json:"blobIds,omitempty"`
	Blobs           []MessageBlob `json:"blobs,omitempty"`
}

// GenericBotSseEventFactChannelTitleUpdate signals that the conversation title
// changed. Title is the new title.
type GenericBotSseEventFactChannelTitleUpdate struct {
	Title string `json:"title"`
}

// GenericBotSseEventFactChannelStatusUpdate signals that the agent declared where
// the work stands. Status is "NEEDS_INPUT" or "COMPLETED" — the badge to show
// beside the conversation.
//
// It answers what the run terminal cannot: an agent stopping to ask a question and
// an agent finishing the job both end on a clean run.done, so they are
// indistinguishable from outside the conversation.
//
// Live-only, and deliberately not replayed on rejoin — a reconnecting client reads
// the current value from ChannelMetadata.ConversationStatus rather than replaying a
// history of superseded verdicts.
type GenericBotSseEventFactChannelStatusUpdate struct {
	Status string `json:"status"`
}

// GenericBotSseEventFactPromptSuggestion is a prediction of what the user is
// likely to send next. Suggestion is a short, ready-to-send line — never empty —
// suitable as input-box placeholder text the user can accept as-is.
type GenericBotSseEventFactPromptSuggestion struct {
	Suggestion string `json:"suggestion"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ToolsetName string      `json:"toolsetName"`
	ToolName    string      `json:"toolName"`
	Parameter   interface{} `json:"parameter"`
	// Reason is the short user-facing rationale the agent supplied via the
	// `_reason` meta property on the tool input. Empty when not supplied.
	Reason string `json:"reason"`
}

// GenericBotSseEventWrapper wraps an SSE event with connection error information
// Used by clients to handle both events and connection errors
type GenericBotSseEventWrapper struct {
	Event           *GenericBotSseEvent `json:"event"`
	ConnectionError error               `json:"connectionError,omitempty"`
}
