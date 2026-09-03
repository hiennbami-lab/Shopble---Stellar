package gmedia

import (
	"shopble/common/comtypes"
	"shopble/config"

	"github.com/spf13/viper"
)

type ConfigMeta struct {
	IsDefault       bool   `mapstructure:"is_default"`
	Bucket          string `mapstructure:"bucket"`
	DefaultFolder   string `mapstructure:"default_folder"`
	CredentialsPath string `mapstructure:"credentials_path"`

	AccessKey       string `mapstructure:"access_key"`
	SecretKey       string `mapstructure:"secret_key"`
	Region          string `mapstructure:"region"`
	InternalBaseUrl string `mapstructure:"internal_base_url"`
	PublicBaseUrl   string `mapstructure:"public_base_url"`
}

type Config struct {
	Local  *ConfigMeta `mapstructure:"local"`
	Google *ConfigMeta `mapstructure:"google"`
	S3     *ConfigMeta `mapstructure:"s3"`
	Minio  *ConfigMeta `mapstructure:"minio"`
}

type (
	ConfigMetaIdx string
)

const (
	ConfigMetaIdxGoogle  ConfigMetaIdx = "google"
	ConfigMetaIdxLocal   ConfigMetaIdx = "local"
	ConfigMetaIdxS3      ConfigMetaIdx = "s3"
	ConfigMetaIdxMinio   ConfigMetaIdx = "minio"
	ConfigMetaIdxDefault ConfigMetaIdx = "default"
)

var (
	vConfig = comtypes.NewSingletonSafe(func() (_ map[ConfigMetaIdx]*ConfigMeta, err error) {
		var (
			conf    Config
			confMap = make(map[ConfigMetaIdx]*ConfigMeta)
		)
		err = viper.UnmarshalKey(config.KeyMediaStorage, &conf)
		if err != nil {
			return
		}
		if conf.Google != nil {
			confMap[ConfigMetaIdxGoogle] = conf.Google
		}
		if conf.S3 != nil {
			confMap[ConfigMetaIdxS3] = conf.S3
		}
		if conf.Minio != nil {
			confMap[ConfigMetaIdxMinio] = conf.Minio
		}
		if conf.Local != nil {
			confMap[ConfigMetaIdxLocal] = conf.Local
		}
		return confMap, err
	})
)

func GetConfig() (map[ConfigMetaIdx]*ConfigMeta, error) {
	return vConfig.Get()
}
