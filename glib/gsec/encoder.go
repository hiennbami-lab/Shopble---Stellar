package gsec

import "encoding/base64"

type Base64Encoder struct {
}

func (b Base64Encoder) EncodeRaw(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func (b Base64Encoder) DecodeRaw(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}
