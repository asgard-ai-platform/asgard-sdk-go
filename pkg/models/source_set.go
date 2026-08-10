package models

type SourceSetDirEntry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"isDir"`
	SizeBytes int64  `json:"sizeBytes"`
	MtimeUnix int64  `json:"mtimeUnix"`
	// Mode is the Unix permission bits (e.g. 420 = 0644).
	Mode uint32 `json:"mode"`
}

type SourceSetPaging struct {
	Index int64 `json:"index"`
	Size  int64 `json:"size"`
	Total int64 `json:"total"`
}

type SourceSetListDirectoryResult struct {
	Entries []SourceSetDirEntry `json:"entries"`
	Paging  *SourceSetPaging    `json:"paging"`
}

type SourceSetStatResult struct {
	Exists    bool   `json:"exists"`
	IsDir     bool   `json:"isDir"`
	SizeBytes int64  `json:"sizeBytes"`
	MtimeUnix int64  `json:"mtimeUnix"`
	Etag      string `json:"etag"`
	// Mode is the Unix permission bits (e.g. 420 = 0644).
	Mode uint32 `json:"mode"`
}

// SourceSetReadMeta holds metadata from the X-Total-Bytes / X-Truncated response
// headers of a volume read.
type SourceSetReadMeta struct {
	// TotalBytes is the full size of the file, independent of any requested range.
	TotalBytes int64
	// Truncated reports that content remains past what was returned — i.e. a
	// limit stopped the read short. An offset read that reaches the end of the
	// file is not truncated.
	Truncated bool
}

type SourceSetWriteFileResult struct {
	BytesWritten int64 `json:"bytesWritten"`
}

type SourceSetCopyResult struct {
	BytesCopied int64 `json:"bytesCopied"`
}
