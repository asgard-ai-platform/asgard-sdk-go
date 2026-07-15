package models

// ChannelHomeDownloadMeta holds metadata from the channel-home download
// response headers: the filename parsed from Content-Disposition and the
// Content-Type mime.
type ChannelHomeDownloadMeta struct {
	FileName string
	MimeType string
}
