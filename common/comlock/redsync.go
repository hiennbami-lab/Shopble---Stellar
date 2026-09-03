package comlock

import (
	"shopble/common/comtypes"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

var (
	vDefaultRedsync = comtypes.NewSingleton(func() *OurRedSync {
		redisClient := GetRedisClient()
		var (
			conf       = GetConfig()
			systempool = goredis.NewPool(redisClient)
		)
		return &OurRedSync{
			Redsync:            redsync.New(systempool),
			defaultLockTimeout: conf.LockTimeout * time.Second,
			defaultRetryDelay:  conf.RetryDelay * time.Second,
		}
	})
)

type OurRedSync struct {
	*redsync.Redsync

	defaultLockTimeout time.Duration
	defaultRetryDelay  time.Duration
}

func (rs *OurRedSync) NewMutex(key string, options ...redsync.Option) *OurMutex {
	return NewMutex(rs.Redsync.NewMutex(key, options...))
}
