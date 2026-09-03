package config

import (
	"os"

	"github.com/spf13/viper"
)

func Init() error {
	defaultPath := os.Getenv(KeyConfigPath)
	setDefaults()
	viper.AutomaticEnv()

	viper.SetConfigFile(defaultPath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	initValues()
	return nil
}

func setDefaults() {
	viper.SetDefault(KeyDebug, true)
	viper.SetDefault(KeyEnv, "dev")
	viper.SetDefault(KeyAppName, "ainow")
	viper.SetDefault(KeyReleaseMode, "dev")
}
