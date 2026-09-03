package gmeta

import (
	"fmt"
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"net/url"
)

type CustomUri string

func (p CustomUri) Query(queryMap comtypes.ISetOrder[string, string]) CustomUri {
	var (
		urlValues = url.Values{}
	)
	for _, v := range queryMap.AsList() {
		urlValues.Set(v.Key, v.Value)
	}
	return p + CustomUri("?"+urlValues.Encode())
}

func (p CustomUri) Param(params ...string) CustomUri {
	var (
		pStr = string(p)
	)
	return CustomUri(fmt.Sprintf(pStr, comutils.ToList(params, func(param string) any {
		return param
	})...))
}

func (p CustomUri) String() string {
	return string(p)
}
