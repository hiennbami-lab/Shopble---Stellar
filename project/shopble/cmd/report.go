package cmd

import (
	"fmt"
	"os"

	"shopble/database"
	"shopble/project/shopble/lib/libstellar"
	"shopble/project/shopble/models"
	"shopble/project/shopble/services/report"

	"github.com/spf13/cobra"
)

var reportCmd = cobra.Command{
	Use:              "report",
	Short:            "Generate the SOW evidence pack (markdown) from stored orders and evidence",
	PersistentPreRun: watchPrerun,
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("out")
		wallet, _ := cmd.Flags().GetString("buyer-wallet")

		db := database.GetDb().DB
		var orders []models.OrderIntent
		q := db.Order("created_at ASC")
		if wallet != "" {
			q = q.Where("buyer_wallet = ?", wallet)
		}
		if err := q.Find(&orders).Error; err != nil {
			fmt.Fprintln(os.Stderr, "đọc order:", err)
			os.Exit(1)
		}

		// Evidence theo thứ tự ledger đóng, không theo lúc capture: bảng nộp phải đọc
		// được như một dòng thời gian của on-chain, không phải thứ tự watcher tình cờ thấy.
		var evidence []models.PaymentEvidence
		if err := db.Order("ledger_close_at ASC, op_id ASC").Find(&evidence).Error; err != nil {
			fmt.Fprintln(os.Stderr, "đọc evidence:", err)
			os.Exit(1)
		}

		md := report.Markdown(libstellar.GetStellarConfig(), orders, evidence)
		if out == "" {
			fmt.Print(md)
			return
		}
		if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "ghi file:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "đã ghi %s (%d order, %d evidence)\n", out, len(orders), len(evidence))
	},
}

func init() {
	rootCmd.AddCommand(&reportCmd)
	reportCmd.Flags().String("out", "", "Write to this file instead of stdout")
	reportCmd.Flags().String("buyer-wallet", "", "Only include orders from this wallet")
}
