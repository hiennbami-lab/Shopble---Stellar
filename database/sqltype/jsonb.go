package sqltype

import (
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strings"

	"shopble/common/comutils"

	"github.com/jackc/pgtype"
)

type Json []byte

func (jsonb Json) Value() (driver.Value, error) {
	if len(jsonb) == 0 {
		return nil, nil
	}
	var jpgtype pgtype.JSONB
	if err := jpgtype.Set([]byte(jsonb)); err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}
	return jpgtype.Value()
}

func (jsonb *Json) Scan(src any) error {
	var jpgtype pgtype.JSONB
	err := jpgtype.Scan(src)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	*jsonb = Json(jpgtype.Bytes)
	return nil
}

func (jsonb *Json) Decode(val any) (err error) {
	if len(*jsonb) == 0 {
		return nil
	}
	var buff = []byte(*jsonb)
	if strings.HasPrefix(string(buff), "\"") {
		buff, err = base64.StdEncoding.DecodeString(strings.Trim(
			string(buff), "\"",
		))
		if err != nil {
			return err
		}
	}
	err = comutils.DecodeJson(buff, val)
	if err != nil {
		return err
	}
	return nil
}
