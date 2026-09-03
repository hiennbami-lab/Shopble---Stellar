package sqltype

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgtype"
)

type BigIntArray[T int64 | uint64] []T

func (ta BigIntArray[T]) Value() (driver.Value, error) {
	if len(ta) == 0 {
		emptyArray := pgtype.Int8Array{Status: pgtype.Present}
		return emptyArray.Value()
	}
	var bigintArray pgtype.Int8Array
	if err := bigintArray.Set(ta); err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}
	return bigintArray.Value()
}

func (ta *BigIntArray[T]) Scan(src any) error {
	var arr pgtype.Int8Array
	if err := arr.Scan(src); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	result := make([]T, len(arr.Elements))
	for idx := range arr.Elements {
		result[idx] = T(arr.Elements[idx].Int)
	}
	*ta = BigIntArray[T](result)
	return nil
}
