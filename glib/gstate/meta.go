package gstate

import (
	"encoding/hex"
	"hash"
)

type StateCode string

func (s StateCode) Hash(method hash.Hash) string {
	method.Write([]byte(s))
	return hex.EncodeToString(method.Sum(nil))
}

func (s StateCode) IsSame(method hash.Hash, hasedState string) bool {
	return s.Hash(method) == hasedState
}

func (s StateCode) String() string {
	return string(s)
}

type StateData struct {
	Identifier string
}
