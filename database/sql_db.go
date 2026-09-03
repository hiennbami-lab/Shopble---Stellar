package database

import "gorm.io/gorm"

type SqlDb struct {
	*gorm.DB
}

func NewDb(db *gorm.DB) *SqlDb {
	return &SqlDb{DB: db}
}
