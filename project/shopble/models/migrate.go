package models

import (
	"fmt"

	"gorm.io/gorm"
)

// AllModels — single source of truth cho auto-migrate.
var AllModels = []any{
	&OrderIntent{},
	&PaymentEvidence{},
	&WatcherCursor{},
}

func AutoMigrate(db *gorm.DB) {
	if err := db.AutoMigrate(AllModels...); err != nil {
		panic(fmt.Errorf("auto-migrate failed: %w", err))
	}
}
