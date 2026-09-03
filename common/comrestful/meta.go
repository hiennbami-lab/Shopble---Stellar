package comrestful

import (
	"io"
)

type Method int

const (
	GET Method = iota
	POST
	PUT
	DELETE
)

type EncoderType int

const (
	JSON EncoderType = iota
	// Body type: map[string]string
	FormURL
	// Body type: map[string]string
	FormMultipart
)

type (
	EncoderOption struct {
		Type EncoderType
		Data any
	}

	RequestOption struct {
		Method Method
		Url    string
		Header Header
		Body   *EncoderOption
	}

	Encoder interface {
		Encode(data any) (io.Reader, error)
	}
)
