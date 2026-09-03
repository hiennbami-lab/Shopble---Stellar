package comcache

import (
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"shopble/config"

	"github.com/spf13/viper"
)

type (
	ConfigRedis struct {
		Address string `mapstructure:"address"`
		Db      int    `mapstructure:"db"`
	}
	Config struct {
		Redis *ConfigRedis `mapstructure:"redis"`
	}
)

var (
	vConfig = comtypes.NewSingleton(func() *Config {
		var (
			conf Config
		)
		comutils.PanicOnError(
			viper.UnmarshalKey(config.KeyCacheDbConnection, &conf))
		return &conf
	})
)

func GetConfig() *Config {
	return vConfig.GetF()
}
