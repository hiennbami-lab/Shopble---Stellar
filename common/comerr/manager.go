package comerr

import (
	"fmt"

	"cloud.google.com/go/logging"
	"github.com/go-errors/errors"
)

type Failure interface {
	Stack() string
	Message() string
	Actual() error
	IsOurError() bool
	Error() string
	Data() AdditionalData
	Severity() logging.Severity
}

func WrapMessage(err error, msg string) *OurError {
	if err == nil {
		err = fmt.Errorf("%s", msg)
	}
	return NewOurError(err, msg)
}

func WrapStack(err error, msg string) *OurError {
	if err == nil {
		err = fmt.Errorf("%s", msg)
	}
	ourError := NewOurError(err, msg)
	ourError.stack = errors.Wrap(ourError.Error(), 1)
	return ourError
}
