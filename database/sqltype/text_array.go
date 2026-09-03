package sqltype

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgtype"
)

type StringArray []string

func (ta StringArray) Value() (driver.Value, error) {
	if len(ta) == 0 {
		emptyArray := pgtype.TextArray{Status: pgtype.Present}
		return emptyArray.Value()
	}
	var textArray pgtype.TextArray
	if err := textArray.Set(ta); err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}
	return textArray.Value()
}

func (ta *StringArray) Scan(src any) error {
	var arr pgtype.TextArray
	if err := arr.Scan(src); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	result := make([]string, len(arr.Elements))
	for idx := range arr.Elements {
		result[idx] = arr.Elements[idx].String
	}
	*ta = StringArray(result)
	return nil
}
