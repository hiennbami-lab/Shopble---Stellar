package cmd

import (
	"shopble/glib/gcmd"

	"github.com/spf13/cobra"
)

var rootCmd = cobra.Command{
	Short: "Shopble — Stellar stablecoin checkout & order reconciliation",
}

func Execute() {
	gcmd.ExecuteRootCmd(&rootCmd)
}
