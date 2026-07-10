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
	MessageThinkingStart    *GenericBotSseEventFactMessage              `json:"messageThinkingStart"`
	MessageThinkingDelta    *GenericBotSseEventFactMessage              `json:"messageThinkingDelta"`
	MessageThinkingComplete *GenericBotSseEventFactMessage              `json:"messageThinkingComplete"`
	ToolCallStart           *GenericBotSseEventFactToolCallStart        `json:"toolCallStart"`
	ToolCallComplete        *GenericBotSseEventFactToolCallComplete     `json:"toolCallComplete"`
	ToolCallConsent         *GenericBotSseEventFactToolCallConsent      `json:"toolCallConsent"`
	CompletionModelUsage    *GenericBotSseEventFactCompletionModelUsage `json:"completionModelUsage"`
	SandboxLaunch           *GenericBotSseEventFactSandboxLaunch        `json:"sandboxLaunch"`
	SandboxReady            *GenericBotSseEventFactSandboxReady         `json:"sandboxReady"`
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
	// ToolUseResultSidecar is the CLI's structured tool-result sidecar (the
	// `tool_use_result` sibling on a user frame) carried verbatim alongside the
	// human-readable ToolCallResult. It exposes structured data the flattened
	// string drops — e.g. TaskCreate → {task:{id,subject}} and TaskUpdate →
	// {taskId,statusChange,...} give a clean task id so a UI can build/track a
	// task list by id (accumulating per id, as Claude Code's Task tools intend)
	// without parsing the result string. nil when the CLI attached none.
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
