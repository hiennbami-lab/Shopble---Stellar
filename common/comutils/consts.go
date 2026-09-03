package comutils

import "github.com/shopspring/decimal"

const (
	DateFormatISO     = "2006-01-02"
	DateTimeFormatISO = "2006-01-02 15:04:05"

	Hex0x                        = "0x"
	StringAlphaNumericCharacters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	DecimalDivisionPrecision = 32
)

var (
	DecimalOneNegative = decimal.NewFromInt(-1)
	DecimalOne         = decimal.NewFromInt(1)
	DecimalTen         = decimal.NewFromInt(10)
)
