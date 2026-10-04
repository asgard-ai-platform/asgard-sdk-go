package models

// ChannelHomeDownloadMeta holds metadata from the channel-home download
// response headers: the filename parsed from Content-Disposition and the
// Content-Type mime.
//
// Deprecated: only returned by the deprecated DownloadChannelHomeFile. Use
// SandboxFsRead for sandbox://<sandboxName>/download-file cards; its
// SandboxFsReadMeta carries TotalBytes and Truncated instead.
type ChannelHomeDownloadMeta struct {
	FileName string
	MimeType string
}
