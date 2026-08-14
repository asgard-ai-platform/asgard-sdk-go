package models

// FileType represents blob file classification returned by EdgeServer.
type FileType string

const (
	FileTypeBinary   FileType = "BINARY"
	FileTypeImage    FileType = "IMAGE"
	FileTypeVideo    FileType = "VIDEO"
	FileTypeAudio    FileType = "AUDIO"
	FileTypeDocument FileType = "DOCUMENT"
)

// Blob represents uploaded blob metadata.
type Blob struct {
	ChannelId string   `json:"channelId"`
	BlobId    string   `json:"blobId"`
	FileType  FileType `json:"fileType"`
	FileName  *string  `json:"fileName"`
	Size      int64    `json:"size"`
	Mime      string   `json:"mime"`
}

// ChannelMetadata describes a channel, returned by the channel-metadata
// endpoint. Title is the conversation title (nil when the channel has none yet);
// RunState is the channel's current run lifecycle (e.g. "IDLE", "RUNNING",
// "ERROR"); LastActivityAt is the last activity time in Unix epoch milliseconds.
type ChannelMetadata struct {
	CustomChannelId string  `json:"customChannelId"`
	Title           *string `json:"title"`
	RunState        string  `json:"runState"`
	LastActivityAt  int64   `json:"lastActivityAt"`
	// LaunchedSandboxes lists the channel's currently-live Sandboxes (Ready and
	// within their shutdown lease). Empty when none are live. A channel may back
	// more than one sandbox, so do not assume a singleton.
	LaunchedSandboxes []LaunchedSandbox `json:"launchedSandboxes"`
}

// LaunchedSandbox describes one live Sandbox backing a channel, so a client can
// open its browser (Neko) handoff or render its working-directory file explorer.
type LaunchedSandbox struct {
	SandboxName          string `json:"sandboxName"`
	SandboxBlueprintName string `json:"sandboxBlueprintName"`
	WorkingDirectory     string `json:"workingDirectory"`
	EditorServerEnabled  bool   `json:"editorServerEnabled"`
	BrowserEnabled       bool   `json:"browserEnabled"`
}

// GenericBotReply is the sync response payload from /message endpoint.
type GenericBotReply struct {
	RequestId              string                  `json:"requestId"`
	Namespace              string                  `json:"namespace"`
	BotProviderName        string                  `json:"botProviderName"`
	CustomChannelId        string                  `json:"customChannelId"`
	Messages               []BufferedMessage       `json:"messages"`
	ErrorDetail            *ErrorDetail            `json:"errorDetail"`
	ToolCallConsentRequest *ToolCallConsentRequest `json:"toolCallConsentRequest,omitempty"`
}

// GenericBotDispatchReply is what Dispatch returns: an acknowledgement that the run
// was accepted, not its result. There is no reply text here by design — the caller
// walked away. Rejoin later with NewChannelStreamer, or read ChannelMetadata for the
// run state.
type GenericBotDispatchReply struct {
	RequestId       string `json:"requestId"`
	CustomChannelId string `json:"customChannelId"`
}

// ToolCallConsentRequest is included in GenericBotReply when the bot requires
// user consent before executing pending tool calls.
type ToolCallConsentRequest struct {
	PendingCalls []PendingToolCall `json:"pendingCalls"`
}
