package gmedia

import (
	"bytes"
	"context"
	"shopble/common/comerr"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type StorageGoogle struct {
	*storage.Client
	bucket        string
	defaultFolder string
	ctx           context.Context
}

func init() {
	RegisterMediaClientLoader(ConfigMetaIdxGoogle, func(cm *ConfigMeta) func(ctx context.Context) (MediaClient, error) {
		return newStorageGoogle(cm)
	})
}

func newStorageGoogle(conf *ConfigMeta) func(ctx context.Context) (_ MediaClient, err error) {
	return func(ctx context.Context) (_ MediaClient, err error) {
		client, err := storage.NewClient(ctx, option.WithCredentialsFile(conf.CredentialsPath))
		if err != nil {
			return
		}
		return &StorageGoogle{
			Client:        client,
			bucket:        conf.Bucket,
			defaultFolder: conf.DefaultFolder,
			ctx:           ctx,
		}, nil
	}
}

func (g *StorageGoogle) ReadMultipleFiles([]MediaMeta) ([]MultipleMediaFileResult, error) {
	return []MultipleMediaFileResult{}, nil
}

func (g *StorageGoogle) UploadFile(file MediaMeta) (_ MediaResult, err error) {
	defer func() {
		_ = g.Close()
	}()
	if file.FileName == "" {
		err = comerr.WrapMessage(comerr.ErrorDataInvalid, "fileName is required")
		return
	}
	folder := g.defaultFolder
	if file.Folder != "" {
		folder = file.Folder
	}
	var (
		filepath = folder + "/" + file.FileName
		writer   = g.Bucket(g.bucket).
				Object(filepath).NewWriter(g.ctx)
	)
	defer func() {
		_ = writer.Close()
	}()
	_, err = io.Copy(writer, file.Reader)
	if err != nil {
		return
	}
	return MediaResult{}, nil
}

func (g *StorageGoogle) GeneratePreviewLink(objectKey string, expireIn time.Duration) (string, error) {
	return "", nil
}

func (g *StorageGoogle) ReadFileBuffer(fileMeta MediaMeta) (_ *bytes.Buffer, err error) {
	return
}

func (g *StorageGoogle) DeleteFile(objectKey string) error { return nil }

func (g *StorageGoogle) Close() (err error) {
	if g.Client == nil {
		return
	}
	_ = g.Client.Close()
	g.Client = nil
	return
}
