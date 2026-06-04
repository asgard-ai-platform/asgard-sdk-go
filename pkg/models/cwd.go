package models

// CwdDownloadMeta holds metadata from the cwd download response headers:
// the filename parsed from Content-Disposition and the Content-Type mime.
type CwdDownloadMeta struct {
	FileName string
	MimeType string
}
