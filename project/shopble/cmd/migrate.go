package cmd

import (
	"shopble/database"
	"shopble/project/shopble/models"

	"github.com/spf13/cobra"
)

var migrateCmd = cobra.Command{
	Use:   "migrate",
	Short: "Run GORM auto-migrate for shopble schema",
	Run: func(cmd *cobra.Command, args []string) {
		database.InitDb()
		models.AutoMigrate(database.GetDb().DB)
	},
}

func init() {
	rootCmd.AddCommand(&migrateCmd)
}
