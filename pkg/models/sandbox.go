package models

type SandboxFsDirEntry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"isDir"`
	SizeBytes int64  `json:"sizeBytes"`
	MtimeUnix int64  `json:"mtimeUnix"`
	Mode      uint32 `json:"mode"`
}

type SandboxFsListResult struct {
	Entries   []SandboxFsDirEntry `json:"entries"`
	Truncated bool                `json:"truncated"`
}

// SandboxFsReadMeta holds metadata from the X-Total-Bytes / X-Truncated response headers.
type SandboxFsReadMeta struct {
	TotalBytes int64
	Truncated  bool
}

type SandboxFsWriteResult struct {
	BytesWritten int64 `json:"bytesWritten"`
}

type SandboxFsStatResult struct {
	Exists    bool   `json:"exists"`
	IsDir     bool   `json:"isDir"`
	SizeBytes int64  `json:"sizeBytes"`
	MtimeUnix int64  `json:"mtimeUnix"`
	Mode      uint32 `json:"mode"`
	Etag      string `json:"etag,omitempty"`
}

type SandboxFsCopyResult struct {
	BytesCopied int64 `json:"bytesCopied"`
}

// SandboxFsWatchEvent is one filesystem change delivered by the fs/watch SSE
// endpoint (GET .../sandbox/{name}/fs/watch). Op is one of
// CREATE / WRITE / REMOVE / RENAME / CHMOD.
type SandboxFsWatchEvent struct {
	Op        string `json:"op"`
	Path      string `json:"path"`
	MtimeUnix int64  `json:"mtimeUnix"`
}

type SandboxHeartbeatResult struct {
	ShutdownAt string `json:"shutdownAt"`
}
