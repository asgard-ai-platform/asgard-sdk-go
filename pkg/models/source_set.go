package models

type SourceSetDirEntry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"isDir"`
	SizeBytes int64  `json:"sizeBytes"`
	MtimeUnix int64  `json:"mtimeUnix"`
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
}

type SourceSetWriteFileResult struct {
	BytesWritten int64 `json:"bytesWritten"`
}
