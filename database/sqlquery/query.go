package sqlquery

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func In[T comparable](colName string, vals []T) clause.Expression {
	anyValList := make([]any, len(vals))
	for idx := range vals {
		anyValList[idx] = vals[idx]
	}
	return clause.IN{
		Column: colName,
		Values: anyValList,
	}
}

func DbSelectForUpdate(db *gorm.DB) *gorm.DB {
	return db.Clauses(clause.Locking{
		Strength: "UPDATE",
	})
}

func OrderDesc(colName string) clause.OrderByColumn {
	return clause.OrderByColumn{
		Column: clause.Column{
			Name: colName,
		},
		Desc: true,
	}
}

func OrderAsc(colName string) clause.OrderByColumn {
	return clause.OrderByColumn{
		Column: clause.Column{
			Name: colName,
		},
	}
}

func Gte(colName string, value any) clause.Expression {
	return clause.Gte{
		Column: clause.Column{
			Name: colName,
		},
		Value: value,
	}
}

func Lte(colName string, value any) clause.Expression {
	return clause.Lte{
		Column: clause.Column{
			Name: colName,
		},
		Value: value,
	}
}

func Lt(colName string, value any) clause.Expression {
	return clause.Lt{
		Column: clause.Column{
			Name: colName,
		},
		Value: value,
	}
}

func ILike(colName, value string) clause.Expression {
	pattern := "%" + value + "%"
	return gorm.Expr("? ILIKE ?", clause.Column{Name: colName}, pattern)
}
