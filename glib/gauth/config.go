package gauth

import (
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"shopble/config"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Issuer       string        `mapstructure:"issuer"`
	TokenTimeout time.Duration `mapstructure:"token_timeout"`
	SecretKey    string        `mapstructure:"secret_key"`
}

var (
	vConfig = comtypes.NewSingleton(func() *Config {
		var (
			conf Config
		)
		comutils.PanicOnError(
			viper.UnmarshalKey(config.KeyJwtConfig, &conf),
		)
		return &conf
	})
)

func GetConfig() *Config {
	return vConfig.GetF()
}
