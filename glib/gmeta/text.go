package gmeta

import (
	"crypto/md5"
	"encoding/base64"
	"strings"
)

type Text string

func (t Text) ToLower() string {
	return strings.ToLower(string(t))
}

func (t Text) TrimSpace() string {
	return strings.TrimSpace(string(t))
}

func (t Text) Normalize() string {
	return strings.TrimSpace(t.ToLower())
}

// Not use for security purpose
type HashableText string

func (t HashableText) ToLower() string {
	return strings.ToLower(string(t))
}

func (t HashableText) Normalize() string {
	return strings.TrimSpace(t.ToLower())
}

func (t HashableText) Md5() string {
	hash := md5.Sum([]byte(t.Normalize()))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
