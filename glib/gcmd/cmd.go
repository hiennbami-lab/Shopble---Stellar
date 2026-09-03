package gcmd

import (
	"fmt"
	"os"

	"shopble/common/comrunner"
	"shopble/common/comutils"
	"shopble/config"

	"github.com/spf13/cobra"
)

func ExecuteRootCmd(cmd *cobra.Command) {
	defer comrunner.SafeClose()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func InitCobra() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	comutils.PanicOnError(config.Init())
	config.InitRandom()
}
