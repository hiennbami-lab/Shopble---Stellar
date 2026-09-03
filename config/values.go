package config

import (
	"fmt"

	"github.com/spf13/viper"
)

var (
	AppName     string
	Env         string
	Debug       bool
	ReleaseMode string
)

func getViperNonEmptyString(key string) string {
	value := viper.GetString(key)
	if value == "" {
		panic(fmt.Errorf("config `%s` must be set`", key))
	}
	return value
}

func initValues() {
	AppName = getViperNonEmptyString(KeyAppName)
	Env = getViperNonEmptyString(KeyEnv)
	Debug = viper.GetBool(KeyDebug)
	ReleaseMode = getViperNonEmptyString(KeyReleaseMode)
}
