package comerr

import (
	"fmt"

	"cloud.google.com/go/logging"
	"github.com/go-errors/errors"
)

type OurError struct {
	isOurError bool
	actual     error
	message    string
	stack      *errors.Error
	data       AdditionalData
	severity   logging.Severity
}

func NewOurError(err error, msg string) *OurError {
	return &OurError{
		isOurError: true,
		actual:     err,
		message:    msg,
		severity:   logging.Error,
		data:       make(AdditionalData),
	}
}

func (o *OurError) Severity() logging.Severity {
	return o.severity
}

func (o *OurError) WithSeverity(severity logging.Severity) *OurError {
	o.severity = severity
	return o
}

func (o *OurError) WithField(key string, value any) *OurError {
	o.data[key] = value
	return o
}

func (o *OurError) WithFields(data AdditionalData) *OurError {
	for key, value := range data {
		o.data[key] = value
	}
	return o
}

func (o *OurError) Error() string {
	return fmt.Sprintf("%s | %s", o.message, o.actual.Error())
}

func (o *OurError) Stack() string {
	var (
		ok         bool
		checkedErr = o
	)
	for {
		if !checkedErr.isOurError {
			return ""
		}
		if checkedErr.stack != nil {
			return checkedErr.stack.ErrorStack()
		}
		if checkedErr, ok = checkedErr.actual.(*OurError); !ok {
			break
		}
	}
	return ""
}

func (o *OurError) Data() AdditionalData {
	var (
		ok         bool
		isInnerErr                = false
		checkedErr                = o
		mergedData AdditionalData = checkedErr.data
	)
	for {
		if !checkedErr.isOurError {
			break
		}
		if checkedErr.data != nil && isInnerErr {
			mergedData = mergedData.Merge(checkedErr.data)
		}
		if checkedErr, ok = checkedErr.actual.(*OurError); ok {
			isInnerErr = true
		} else {
			break
		}
	}
	return mergedData
}

func (o *OurError) Message() string {
	return o.message
}

func (o *OurError) IsOurError() bool {
	return o.isOurError
}

func (o *OurError) Actual() error {
	return o.actual
}

type OurErrorCode struct {
	code     string
	ourError *OurError
}

func NewOurErrorCode(code string) OurErrorCode {
	return OurErrorCode{
		code:     code,
		ourError: NewOurError(fmt.Errorf("%s", code), code),
	}
}

func (oc *OurErrorCode) Wrap(err error) *OurErrorCode {
	if err != nil {
		oc.ourError.actual = err
	}
	return oc
}

func (oc *OurErrorCode) Code() string {
	return oc.code
}

func (oc OurErrorCode) Error() string {
	return oc.ourError.Error()
}

func (oc OurErrorCode) Stack() string {
	return oc.ourError.Stack()
}

func (oc OurErrorCode) Message() string {
	return oc.ourError.Message()
}

func (oc OurErrorCode) Actual() error {
	return oc.ourError.Actual()
}

func (oc OurErrorCode) IsOurError() bool {
	return oc.ourError.IsOurError()
}

func (oc OurErrorCode) Severity() logging.Severity {
	return oc.ourError.severity
}

func (oc OurErrorCode) Data() AdditionalData {
	return oc.ourError.Data()
}
