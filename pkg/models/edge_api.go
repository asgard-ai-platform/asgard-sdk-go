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

// ToolCallConsentRequest is included in GenericBotReply when the bot requires
// user consent before executing pending tool calls.
type ToolCallConsentRequest struct {
	PendingCalls []PendingToolCall `json:"pendingCalls"`
}
