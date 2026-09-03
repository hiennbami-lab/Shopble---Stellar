package gmsgqueue

import (
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"shopble/config"

	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

type (
	JetstreamConfig struct {
		Url    string `mapstructure:"url"`
		Stream string `mapstructure:"stream"`
	}
)
type MsgQueueConfig struct {
	JetstreamConfig *JetstreamConfig `mapstructure:"jetstream"`
}

var (
	vConfig = comtypes.NewSingleton(func() *MsgQueueConfig {
		var (
			conf MsgQueueConfig
		)
		comutils.PanicOnError(
			viper.UnmarshalKey(config.KeyMsgQueue, &conf),
		)
		return &conf
	})

	vJetstreamCtx = comtypes.NewSingleton(func() (_ nats.JetStreamContext) {
		conf := GetConfig()
		jsConn, err := nats.Connect(conf.JetstreamConfig.Url)
		comutils.PanicOnError(err)
		jetstreamCtx, err := jsConn.JetStream()
		comutils.PanicOnError(err)
		return jetstreamCtx
	})
)

func GetConfig() *MsgQueueConfig {
	return vConfig.GetF()
}

func GetJsCtx() nats.JetStreamContext {
	return vJetstreamCtx.GetF()
}

func InitMsgQueue() {
	_ = GetJsCtx()
}
