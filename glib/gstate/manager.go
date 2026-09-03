package gstate

import (
	"shopble/common/comutils"
)

func GenerateCode() (code StateCode, err error) {
	codeStr, err := comutils.HexRandom(32)
	if err != nil {
		return
	}
	return StateCode(codeStr), err
}
