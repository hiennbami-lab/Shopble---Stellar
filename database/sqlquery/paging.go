package sqlquery

import (
	"shopble/glib/gmeta"

	"gorm.io/gorm"
)

const (
	PagingEmptyLimit = 500
)

func DbPaging(db *gorm.DB, paging *gmeta.Paging) *gorm.DB {
	if paging.Page <= 0 {
		paging.Page = 1
	}
	if paging.Limit <= 0 {
		paging.Limit = 20
	}
	paging.Offset = (paging.Page - 1) * paging.Limit
	if paging.Limit > 0 {
		db = db.Limit(paging.Limit)
	} else {
		db = db.Limit(PagingEmptyLimit)
	}
	if paging.Offset > 0 {
		db = db.Offset(paging.Offset)
		// } else if paging.BeforeID > 0 {
		// 	db = db.Where(Lt(sqlfields.CommonColID, paging.BeforeID))
		// } else if paging.AfterID > 0 {
		// 	db = db.Where(Gt(sqlfields.CommonColID, paging.AfterID))
	}
	return db
}
