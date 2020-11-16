package storage

import (
	"context"
	"io"
	"strconv"
)

type Metadata map[string]string

const (
	propPrefix              = "X-Oss-Meta-"
	metadataModifyTimestamp = "Modify-Time"
	metadataFilename        = "Filename"
	metadataFileSize        = "File-Size"
)

func (md Metadata) ModTime() int64 {
	if ts, ok := md[propPrefix+metadataModifyTimestamp]; ok {
		if n, err := strconv.ParseInt(ts, 10, 64); err == nil {
			return n
		}
	}

	return 0
}

func (md Metadata) Filename() string {
	if v, ok := md[propPrefix+metadataFilename]; ok {
		return v
	}

	return ""
}

func (md Metadata) FileSize() int {
	if v, ok := md[propPrefix+metadataFileSize]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}

	return 0
}

func (md Metadata) SetModTime(ts int64) {
	md[metadataModifyTimestamp] = strconv.Itoa(int(ts))
}

func (md Metadata) SetFilename(filename string) {
	md[metadataFilename] = filename
}

func (md Metadata) SetFileSize(size int) {
	md[metadataFileSize] = strconv.Itoa(size)
}

type Item struct {
	Filename  string   `json:"filename"`
	ObjectKey string   `json:"object_key"`
	FileSize  int64    `json:"file_size"`
	Metadata  Metadata `json:"metadata"`
}

type Uploader interface {
	Exists(key string) bool
	Metadata(key string) Metadata
	ListObject(prefix string) chan *Item
	Upload(ctx context.Context, item *Item, reader io.Reader, metadata Metadata) error
	Download(ctx context.Context, item *Item, w io.Writer) error
}
