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
	RunInit          *GenericBotSseEventFactRunInit          `json:"runInit"`
	RunDone          *GenericBotSseEventFactRunDone          `json:"runDone"`
	RunError         *GenericBotSseEventFactRunError         `json:"runError"`
	ProcessStart     *GenericBotSseEventFactProcessStart     `json:"processStart"`
	ProcessComplete  *GenericBotSseEventFactProcessComplete  `json:"processComplete"`
	MessageStart     *GenericBotSseEventFactMessage          `json:"messageStart"`
	MessageDelta     *GenericBotSseEventFactMessage          `json:"messageDelta"`
	MessageComplete  *GenericBotSseEventFactMessage          `json:"messageComplete"`
	ToolCallStart    *GenericBotSseEventFactToolCallStart    `json:"toolCallStart"`
	ToolCallComplete *GenericBotSseEventFactToolCallComplete `json:"toolCallComplete"`
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
}

// GenericBotSseEventFactToolCallComplete is emitted when a tool call completes
type GenericBotSseEventFactToolCallComplete struct {
	ProcessId      string      `json:"processId"`
	CallSeq        int         `json:"callSeq"`
	ToolCall       ToolCall    `json:"toolCall"`
	ToolCallResult interface{} `json:"toolCallResult"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ToolsetName string      `json:"toolsetName"`
	ToolName    string      `json:"toolName"`
	Parameter   interface{} `json:"parameter"`
}

// GenericBotSseEventWrapper wraps an SSE event with connection error information
// Used by clients to handle both events and connection errors
type GenericBotSseEventWrapper struct {
	Event           *GenericBotSseEvent `json:"event"`
	ConnectionError error               `json:"connectionError,omitempty"`
}
