package sqltype

import (
	"database/sql/driver"
	"fmt"
	"shopble/glib/gmeta"

	"github.com/jackc/pgtype"
)

type NormalizedText gmeta.Text

func (t NormalizedText) Value() (driver.Value, error) {
	var text pgtype.Text
	if err := text.Set(gmeta.Text(t).Normalize()); err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}
	return text.Value()
}

func (t *NormalizedText) Scan(src any) error {
	var text pgtype.Text
	if err := text.Scan(src); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	*t = (NormalizedText)(text.String)
	return nil
}

type Text gmeta.Text

func (t Text) Value() (driver.Value, error) {
	var text pgtype.Text
	if err := text.Set(gmeta.Text(t).TrimSpace()); err != nil {
		return nil, fmt.Errorf("set: %w", err)
	}
	return text.Value()
}

func (t *Text) Scan(src any) error {
	var text pgtype.Text
	if err := text.Scan(src); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	*t = (Text)(text.String)
	return nil
}
