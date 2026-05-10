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

type SandboxHeartbeatResult struct {
	ShutdownAt string `json:"shutdownAt"`
}
