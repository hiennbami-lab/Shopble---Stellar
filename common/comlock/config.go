package comlock

import (
	"time"

	"shopble/common/comrunner"
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"shopble/config"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

type (
	RedisConfig struct {
		Address string `mapstructure:"address"`
		DB      int    `mapstructure:"db"`
	}
	Config struct {
		Redis       *RedisConfig  `mapstructure:"redis"`
		LockTimeout time.Duration `mapstructure:"lock_timeout"`
		RetryDelay  time.Duration `mapstructure:"retry_delay"`
	}
)

var (
	vConfig = comtypes.NewSingleton(func() *Config {
		var (
			conf Config
		)
		comutils.PanicOnError(
			viper.UnmarshalKey(config.KeyLockDbConnection, &conf),
		)
		return &conf
	})
	vRedisClient = comtypes.NewSingleton(func() *redis.Client {
		var (
			conf = GetConfig()
		)
		conn := redis.NewClient(&redis.Options{
			Addr: conf.Redis.Address,
			DB:   conf.Redis.DB,
		})
		comrunner.RegisterRootCloser(func() {
			_ = conn.Close()
		})
		return conn
	})
)

func GetConfig() *Config {
	return vConfig.GetF()
}

func GetRedisClient() *redis.Client {
	return vRedisClient.GetF()
}

func Init() {
	_ = GetRedisClient()
}
