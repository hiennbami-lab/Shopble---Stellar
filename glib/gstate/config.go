package gstate

import (
	"crypto/hmac"
	"crypto/sha256"
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"hash"

	"github.com/spf13/viper"
)

type Config struct {
	SecretKey  string    `mapstructure:"secret_key"`
	HashMethod hash.Hash `mapstructure:"-"`
}

var (
	vConfig = comtypes.NewSingleton(func() *Config {
		var (
			conf Config
		)
		comutils.PanicOnError(
			viper.UnmarshalKey("", &conf),
		)
		conf.HashMethod = hmac.New(sha256.New, []byte(conf.SecretKey))
		return &conf
	})
)

func GetConfig() *Config {
	return vConfig.GetF()
}
