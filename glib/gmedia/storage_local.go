package gmedia

import (
	"bytes"
	"context"
	"shopble/common/comerr"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type StorageLocal struct {
	bucket string
}

func init() {
	RegisterMediaClientLoader(ConfigMetaIdxLocal, func(cm *ConfigMeta) func(ctx context.Context) (MediaClient, error) {
		return newStorageLocal(cm)
	})
}

func newStorageLocal(conf *ConfigMeta) func(ctx context.Context) (_ MediaClient, err error) {
	return func(ctx context.Context) (_ MediaClient, err error) {
		return &StorageLocal{
			bucket: conf.Bucket,
		}, nil
	}
}

func (g *StorageLocal) ReadMultipleFiles([]MediaMeta) ([]MultipleMediaFileResult, error) {
	return []MultipleMediaFileResult{}, nil
}

func (g *StorageLocal) GeneratePreviewLink(objectKey string, expireIn time.Duration) (string, error) {
	return "", nil
}

func (l *StorageLocal) UploadFile(file MediaMeta) (_ MediaResult, err error) {
	if file.FileName == "" {
		err = comerr.WrapMessage(comerr.ErrorDataInvalid, "fileName is required")
		return
	}
	folder := l.bucket
	if file.Folder != "" {
		folder = file.Folder
	}
	var (
		folderFile = folder + "/" + file.FileName
		dir        = filepath.Dir(folderFile)
	)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return
	}
	osFile, err := os.Create(folderFile)
	if err != nil {
		return
	}
	defer osFile.Close()
	_, err = io.Copy(osFile, file.Reader)
	if err != nil {
		return
	}
	return MediaResult{}, nil
}

func (l *StorageLocal) ReadFileBuffer(fileMeta MediaMeta) (_ *bytes.Buffer, err error) {
	parsedUrl, err := url.Parse(fileMeta.FileName)
	if err != nil {
		return
	}
	file, err := os.Open(parsedUrl.Host + parsedUrl.Path)
	if err != nil {
		return
	}
	defer file.Close()
	var (
		buffer bytes.Buffer
	)
	_, err = io.Copy(&buffer, file)
	if err != nil {
		return
	}
	return &buffer, nil
}

func (l *StorageLocal) DeleteFile(objectKey string) error { return nil }

func (l *StorageLocal) Close() (err error) {
	return nil
}
