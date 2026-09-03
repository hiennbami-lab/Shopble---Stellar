package gmedia

import "io"

type MediaMeta struct {
	io.Reader
	Folder      string
	FileName    string
	ContentType string
}

type MultipleMediaFileResult struct {
	Id      string
	Content string
}

type MediaResult struct {
	PreviewLink string
	Bucket      string
	ObjectKey   string
}

type MediaUrl string
