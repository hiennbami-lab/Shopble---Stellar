package gsec

import (
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"shopble/config"

	"github.com/spf13/viper"
)

type Config struct {
	SecretKey string `mapstructure:"secret_key"`
}

var (
	vConfig = comtypes.NewSingleton(func() *Config {
		var (
			conf Config
		)
		comutils.PanicOnError(
			viper.UnmarshalKey(config.KeySystemSecConfig, &conf),
		)
		return &conf
	})
)

func GetConfig() *Config {
	return vConfig.GetF()
}
