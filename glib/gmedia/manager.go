package gmedia

import (
	"bytes"
	"context"
	"shopble/common/comtypes"
	"time"
)

type (
	MediaClient interface {
		UploadFile(fileMeta MediaMeta) (MediaResult, error)
		GeneratePreviewLink(string, time.Duration) (string, error)
		ReadFileBuffer(fileMeta MediaMeta) (*bytes.Buffer, error)
		ReadMultipleFiles([]MediaMeta) ([]MultipleMediaFileResult, error)
		DeleteFile(objectKey string) error
		Close() error
	}
)

var (
	clientLoader map[ConfigMetaIdx]func(*ConfigMeta) func(ctx context.Context) (MediaClient, error) = make(map[ConfigMetaIdx]func(*ConfigMeta) func(ctx context.Context) (MediaClient, error))
)

func RegisterMediaClientLoader(idx ConfigMetaIdx, loader func(*ConfigMeta) func(ctx context.Context) (MediaClient, error)) {
	_, ok := clientLoader[idx]
	if ok {
		panic("this client loader is already registered")
	}
	clientLoader[idx] = loader
}

var (
	vCLientLoader = comtypes.NewSingletonSafe(func() (map[ConfigMetaIdx]func(ctx context.Context) (MediaClient, error), error) {
		var (
			clientMap = make(map[ConfigMetaIdx]func(ctx context.Context) (MediaClient, error))
		)
		conf, err := GetConfig()
		if err != nil {
			return nil, err
		}
		for key, confMeta := range conf {
			client := clientLoader[key](confMeta)
			clientMap[key] = client
			if confMeta.IsDefault {
				clientMap[ConfigMetaIdxDefault] = client
			}
		}
		return clientMap, nil
	})
)

func GetClient(ctx context.Context, platform ...ConfigMetaIdx) (client MediaClient, err error) {
	clientGetter, err := vCLientLoader.Get()
	if err != nil {
		return
	}
	if len(platform) <= 0 {
		return clientGetter[ConfigMetaIdxDefault](ctx)
	}
	return clientGetter[platform[0]](ctx)
}

func GetGoogleClient(ctx context.Context) (MediaClient, error) {
	return GetClient(ctx, ConfigMetaIdxGoogle)
}
