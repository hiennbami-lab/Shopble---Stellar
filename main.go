package main

import (
	"shopble/config"
	"shopble/glib/gcmd"
	"shopble/maincmd"
	"os"
)

func main() {
	if os.Getenv(config.KeyConfigFree) == "" {
		gcmd.InitCobra()
	}
	maincmd.Execute()
}
