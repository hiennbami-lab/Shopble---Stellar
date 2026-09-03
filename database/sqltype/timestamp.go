package sqltype

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/jackc/pgtype"
)

type Time time.Time

func (t Time) Value() (driver.Value, error) {
	tpgtype := pgtype.Timestamptz{
		Time:   time.Time(t),
		Status: pgtype.Present,
	}
	return tpgtype.Value()
}

func (t *Time) Scan(src any) error {
	tpgtype := pgtype.Timestamptz{}
	if err := tpgtype.Scan(src); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	*t = Time(tpgtype.Time)
	return nil
}
