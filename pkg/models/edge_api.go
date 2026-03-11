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

// GenericBotReply is the sync response payload from /message endpoint.
type GenericBotReply struct {
	RequestId       string            `json:"requestId"`
	Namespace       string            `json:"namespace"`
	BotProviderName string            `json:"botProviderName"`
	CustomChannelId string            `json:"customChannelId"`
	Messages        []BufferedMessage `json:"messages"`
	ErrorDetail     *ErrorDetail      `json:"errorDetail"`
}
