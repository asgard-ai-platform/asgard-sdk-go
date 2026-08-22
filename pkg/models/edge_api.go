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

// Blob represents uploaded blob metadata, as returned by the upload endpoint.
type Blob struct {
	ChannelId string   `json:"channelId"`
	BlobId    string   `json:"blobId"`
	FileType  FileType `json:"fileType"`
	FileName  *string  `json:"fileName"`
	Size      int64    `json:"size"`
	Mime      string   `json:"mime"`
}

// MessageBlob is one attachment as it appears ON a message, carrying what a
// client needs to render it: FileName for the chip's label, FileType and Mime to
// choose an image preview over a document chip, Size for the caption. FileName
// is nil when the upload carried no name — distinct from an empty one, so a
// client can substitute its own label rather than showing a blank.
//
// Deliberately NOT the Blob above, which it otherwise duplicates: Blob has a
// ChannelId, and this shape does not. A relay that decodes a frame and re-encodes
// it would otherwise emit a bogus "channelId":"" that was never on the wire.
type MessageBlob struct {
	BlobId   string   `json:"blobId"`
	FileType FileType `json:"fileType"`
	FileName *string  `json:"fileName"`
	Size     int64    `json:"size"`
	Mime     string   `json:"mime"`
}

// ChannelMetadata describes a channel, returned by the channel-metadata
// endpoint. Title is the conversation title (nil when the channel has none yet);
// RunState is the channel's current run lifecycle (e.g. "IDLE", "RUNNING",
// "ERROR"); LastActivityAt is the last activity time in Unix epoch milliseconds.
type ChannelMetadata struct {
	CustomChannelId string  `json:"customChannelId"`
	Title           *string `json:"title"`
	RunState        string  `json:"runState"`
	// ConversationStatus is the agent's own verdict on where the work stands —
	// "NEEDS_INPUT" or "COMPLETED" — and nil when it has not judged (it is working,
	// or it ended a turn without saying).
	//
	// Read it alongside RunState, not instead of it: RunState says whether a run is
	// in flight, this says whether the user is needed. It cannot be derived from the
	// stream, because an agent that stops to ask a question ends its turn on a clean
	// run.done, identical on the wire to one that finished the job.
	//
	// This is the value to render after opening a conversation or when listing
	// several; while a conversation is streaming, the live changes arrive as
	// SseEventTypeChannelStatusUpdate instead (that event is never replayed on
	// rejoin, which is why this snapshot exists).
	ConversationStatus *string `json:"conversationStatus"`
	LastActivityAt     int64   `json:"lastActivityAt"`
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
