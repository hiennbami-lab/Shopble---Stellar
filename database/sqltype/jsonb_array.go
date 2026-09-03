package sqltype

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgtype"
)

type JsonArray []Json

func (ja JsonArray) Value() (driver.Value, error) {
	if len(ja) == 0 {
		return nil, nil
	}
	var jpgtype pgtype.JSONBArray
	tempArr := make([][]byte, len(ja))
	for idx := range ja {
		tempArr[idx] = ja[idx]
	}
	if err := jpgtype.Set(tempArr); err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}
	return jpgtype.Value()
}

func (ja *JsonArray) Scan(src any) error {
	var jpgtype pgtype.JSONBArray
	if err := jpgtype.Scan(src); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	for _, elem := range jpgtype.Elements {
		*ja = append(*ja, Json(elem.Bytes))
	}
	return nil
}
