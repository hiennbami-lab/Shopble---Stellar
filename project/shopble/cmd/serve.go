package cmd

import (
	"fmt"
	"net/http"
	"time"

	"shopble/api"
	"shopble/common/comrunner"
	shopbleapi "shopble/project/shopble/services/api"
	"shopble/project/shopble/services/watcher"

	"github.com/spf13/cobra"
)

// serve — HTTP API và watcher trong CÙNG một process.
//
// Lý do tồn tại: một VPS (hoặc bất kỳ free tier nào) cho bạn một service. Chạy `api http`
// và `watch` như hai process nghĩa là hai unit phải cùng sống, cùng chết, cùng được giám sát.
// comrunner.NewSession vốn nhận nhiều service và đã có sẵn một đường tắt máy chung, nên
// gộp lại không thêm cơ chế gì mới — chỉ bớt đi một thứ phải trông.
//
// Vẫn giữ `api http` và `watch` riêng: khi cần scale, watcher phải chạy MỘT bản duy nhất
// còn API thì chạy nhiều bản.
var serveCmd = cobra.Command{
	Use:              "serve",
	Short:            "Run the HTTP API and the payment watcher in one process",
	PersistentPreRun: apiPrerun,
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		stream, _ := cmd.Flags().GetString("stream")
		interval, _ := cmd.Flags().GetInt("interval")
		noChain, _ := cmd.Flags().GetBool("no-chain")

		router := api.New()
		shopbleapi.InitRouter(router)

		comrunner.RunSession(comrunner.NewSession(
			comrunner.NewHttpService(&http.Server{
				Handler: router,
				Addr:    fmt.Sprintf("%s:%d", host, port),
			}),
			watcher.New(stream, time.Duration(interval)*time.Second, !noChain),
		))
	},
}

func init() {
	rootCmd.AddCommand(&serveCmd)
	serveCmd.Flags().String("host", "0.0.0.0", "Host to bind")
	serveCmd.Flags().Int("port", 8080, "Port to bind")
	serveCmd.Flags().String("stream", "payments", "Cursor stream name")
	serveCmd.Flags().Int("interval", 3, "Seconds to wait after catching up before polling again")
	serveCmd.Flags().Bool("no-chain", false, "Do not write verdicts to the Soroban contract")
}
