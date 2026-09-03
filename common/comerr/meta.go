package comerr

import (
	"fmt"
	"shopble/common/comtypes"
)

type AdditionalData comtypes.KeyValue[string, any]

func (d AdditionalData) Merge(data AdditionalData) AdditionalData {
	return AdditionalData(comtypes.KeyValue[string, any](d).Merge(data))
}

func (a AdditionalData) String() string {
	dataStr := ""
	for key, value := range a {
		dataStr += fmt.Sprintf("%s: %v\n", key, value)
	}
	return dataStr
}
