package comutils

import (
	"errors"
	"strings"

	"shopble/common/comerr"

	"gorm.io/gorm"
)

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func IsSameError(err error, target comerr.OurErrorCode) bool {
	typedError, ok := err.(comerr.Failure)
	if !ok {
		return false
	}
	for {
		if !typedError.IsOurError() {
			return false
		}
		if typedError.Message() == target.Code() {
			return true
		}
		typedError, ok = typedError.Actual().(comerr.Failure)
		if !ok {
			return false
		}
	}
}

func ErrorUnwrapRoot(err error) error {
	for {
		wrappedErr := errors.Unwrap(err)
		if wrappedErr == nil {
			break
		}
		err = wrappedErr
	}
	return err
}

func IsDbErrorNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

const (
	// Reference: https://www.postgresql.org/docs/13/errcodes-appendix.html
	PgSqlErrorCodeDuplicateEntry = "23505"
)

func IsErrorDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), PgSqlErrorCodeDuplicateEntry)
}

func IsDbError(err error) bool {
	if err == nil {
		return false
	}
	if IsDbErrorNotFound(err) {
		return false
	}
	return true
}
