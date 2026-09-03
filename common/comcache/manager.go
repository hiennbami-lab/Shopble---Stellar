package comcache

import (
	"context"
	"shopble/common/comtypes"
	"time"
)

var (
	loaderMap map[string]func(*Config) ICacheManager = make(map[string]func(*Config) ICacheManager)
)

func RegisterCacheHandler(cacheIdx string, loader func(*Config) ICacheManager) {
	_, ok := loaderMap[cacheIdx]
	if ok {
		panic("cache loader already registered! loader only registered once")
	}
	loaderMap[cacheIdx] = loader
}

type (
	Options struct {
		Timeout time.Duration
	}

	IEncoder interface {
		Encode(v any) ([]byte, error)
		Decode(data []byte, v any) error
	}

	ICacheManager interface {
		RLock()
		RUnlock()
		Lock()
		Unlock()

		Exists(ctx context.Context, key string) (bool, error)
		Get(ctx context.Context, key string, result any) error
		Set(ctx context.Context, key string, value any, timeout time.Duration) error
		SetEx(ctx context.Context, key string, value any, options Options) error
		Delete(ctx context.Context, key string) error
	}
)

var (
	vCacheGetter = comtypes.NewSingleton(func() (_ map[string]ICacheManager) {
		var (
			conf  = GetConfig()
			dbMap = make(map[string]ICacheManager)
		)
		for key, loader := range loaderMap {
			dbMap[key] = loader(conf)
		}
		return dbMap
	})
)

func Init(dataDumper IEncoder) {
	_ = GetDefaultCache()
}

func GetDefaultCache() ICacheManager {
	return vCacheGetter.GetF()["redis"]
}
