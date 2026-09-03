package cmd

import (
	"time"

	"shopble/common/comrunner"
	"shopble/database"
	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/services/watcher"

	"github.com/spf13/cobra"
)

func watchPrerun(cmd *cobra.Command, args []string) {
	database.InitDb()
	// Cùng lý do như api: nạp Stellar config ngay để cổng chặn mainnet chết lúc boot,
	// không phải khi record đầu tiên tới.
	libstellar.GetStellarConfig()
}

var watchCmd = cobra.Command{
	Use:              "watch",
	Short:            "Stream Horizon payments to the destination account and reconcile orders",
	PersistentPreRun: watchPrerun,
	Run: func(cmd *cobra.Command, args []string) {
		stream, _ := cmd.Flags().GetString("stream")
		interval, _ := cmd.Flags().GetInt("interval")
		w := watcher.New(stream, time.Duration(interval)*time.Second)
		comrunner.RunSession(comrunner.NewSession(w))
	},
}

func init() {
	rootCmd.AddCommand(&watchCmd)
	watchCmd.Flags().String("stream", "payments", "Cursor stream name")
	watchCmd.Flags().Int("interval", 3, "Seconds to wait after catching up before polling again")
}
