package cmd

import (
	"shopble/database"
	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/services/api"

	"github.com/spf13/cobra"
)

func apiPrerun(cmd *cobra.Command, args []string) {
	database.InitDb()
	// Nạp Stellar config NGAY ở startup. Singleton này lazy, nên nếu để nó tự nạp ở
	// request đầu tiên thì cổng chặn mainnet chỉ nổ bên trong handler, bị RecoverPanic
	// nuốt thành 500 — trong khi /health và các route đọc vẫn xanh. Ops sẽ thấy một app
	// "đang chạy" với config trỏ vào public network. Sai config phải chết lúc boot.
	libstellar.GetStellarConfig()
}

var apiCmd = cobra.Command{
	Use:   "api",
	Short: "API command",
}

var apiHttpCmd = cobra.Command{
	Use:              "http",
	Short:            "Serve HTTP API service",
	PersistentPreRun: apiPrerun,
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		api.StartHttpSvc(host, port)
	},
}

func init() {
	rootCmd.AddCommand(&apiCmd)

	apiCmd.AddCommand(&apiHttpCmd)
	apiHttpCmd.Flags().String("host", "0.0.0.0", "Host to bind")
	apiHttpCmd.Flags().Int("port", 8080, "Port to bind")
}
