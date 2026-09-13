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

// SandboxBrowserSession is what a client needs to drive the sandbox browser
// itself — render the WebRTC stream and forward keyboard/mouse — instead of
// opening Neko's own UI in a tab (which is what GenerateSandboxBrowserOpenUrl
// is for). Everything past this handshake is the Neko protocol: connect the
// WebSocket with the token, then negotiate WebRTC over it; the server is the
// offerer.
//
// Token is a Neko session token, not an Asgard one. It grants what a member of
// that sandbox's browser can do — the same thing the user could already do
// through the open-url flow — and it dies with the sandbox pod.
type SandboxBrowserSession struct {
	// WsUrl is the absolute wss:// URL of the sandbox's Neko WebSocket. The
	// token is NOT embedded: append it as `?token=<token>` when connecting.
	// Browsers cannot set headers on a WebSocket, so the query parameter is the
	// only option there; keeping it out of this field also stops the credential
	// from being duplicated into anything that logs the URL.
	WsUrl string `json:"wsUrl"`
	Token string `json:"token"`
}
